package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"akguard/internal/config"
	"akguard/internal/handler"
	"akguard/internal/middleware"
	"akguard/internal/model"
)

func main() {
	dbPath := "/data/akguard.db"
	if os.Getenv("AKGUARD_DEV") == "1" {
		dbPath = "akguard.db"
	}

	// 初始化 SQLite
	if err := model.InitDB(dbPath); err != nil {
		log.Fatal("failed to initialize database:", err)
	}

	// 从数据库加载配置
	cfg := config.LoadConfig()
	parsedNets := config.ParseCIDRs(cfg.Auth.InternalNets)

	state := &config.AppState{
		Config:     cfg,
		ParsedNets: parsedNets,
		Sessions:   config.NewSessionStore(),
		Attempts:   config.NewAttemptTracker(),
	}

	otpStore := middleware.NewOTPStore()

	// 端口：默认 3000，可通过环境变量覆盖
	addr := cfg.Server.Listen // ":3000"
	if port := os.Getenv("AKGUARD_LISTEN_PORT"); port != "" {
		addr = ":" + port
	}

	devMode := os.Getenv("AKGUARD_DEV") == "1"

	mux := http.NewServeMux()

	// 核心鉴权端点
	mux.HandleFunc("/verify", handler.Verify(state))

	// API 接口
	mux.HandleFunc("/api/auth/login", handler.AuthLogin(state, otpStore))
	mux.HandleFunc("/api/auth/logout", handler.AuthLogout(state))
	mux.HandleFunc("/api/admin/login", handler.AdminLogin(state, otpStore))
	mux.HandleFunc("/api/admin/logout", handler.AdminLogout(state))
	mux.HandleFunc("/api/otp/send", handler.SendOTP(state, otpStore))

	mux.HandleFunc("/api/config", onlyMethod(handler.GetConfig(state), "GET"))
	mux.HandleFunc("/api/login-methods", onlyMethod(handler.GetLoginMethods(state), "GET"))
	mux.HandleFunc("/api/config/auth-password", onlyMethod(handler.UpdateAuthPassword(state), "PUT"))
	mux.HandleFunc("/api/config/admin-password", onlyMethod(handler.UpdateAdminPassword(state), "PUT"))
	mux.HandleFunc("/api/config/bark", onlyMethod(handler.UpdateBark(state), "PUT"))
	mux.HandleFunc("/api/config/test-bark", onlyMethod(handler.TestBark(state), "POST"))
	mux.HandleFunc("/api/config/confirm-bark", onlyMethod(handler.ConfirmBark(state), "POST"))
	mux.HandleFunc("/api/config/networks", onlyMethod(handler.UpdateNetworks(state), "PUT"))
	mux.HandleFunc("/api/config/domains", onlyMethod(handler.UpdateDomains(state), "PUT"))
	mux.HandleFunc("/api/config/default-policy", onlyMethod(handler.UpdateDefaultPolicy(state), "PUT"))
	mux.HandleFunc("/api/config/admin-login-methods", onlyMethod(handler.UpdateAdminLoginMethods(state), "PUT"))
	mux.HandleFunc("/api/config/auth-login-methods", onlyMethod(handler.UpdateAuthLoginMethods(state), "PUT"))
	mux.HandleFunc("/api/config/auth-ban", onlyMethod(handler.UpdateAuthBanConfig(state), "PUT"))
	mux.HandleFunc("/api/config/admin-ban", onlyMethod(handler.UpdateAdminBanConfig(state), "PUT"))
	mux.HandleFunc("/api/config/site-title", onlyMethod(handler.UpdateSiteTitle(state), "PUT"))
	mux.HandleFunc("/api/blacklist", onlyMethod(handler.GetBlacklist(state), "GET"))
	mux.HandleFunc("/api/blacklist/add", onlyMethod(handler.AddBlacklist(state), "POST"))

	mux.HandleFunc("/api/dashboard", onlyMethod(handler.GetDashboard(state), "GET"))
	mux.HandleFunc("/api/logs/access/stats", onlyMethod(handler.GetAccessLogStats(state), "GET"))
	mux.HandleFunc("/api/logs/access", onlyMethod(handler.GetAccessLogs(state), "GET"))
	mux.HandleFunc("/api/logs/audit", onlyMethod(handler.GetAuditLogs(state), "GET"))
	mux.HandleFunc("/api/sessions", onlyMethod(handler.GetSessions(state), "GET"))

	// 静态文件
	distDir := "frontend/dist"
	if d := os.Getenv("AKGUARD_DIST"); d != "" {
		distDir = d
	}
	fs := http.FileServer(http.Dir(distDir))

	var rootHandler http.Handler = mux
	rootHandler = middleware.CORS(devMode)(rootHandler)

	deleteSessionHandler := handler.DeleteSession(state)
	deleteBlacklistHandler := handler.DeleteBlacklist(state)

	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" && strings.HasPrefix(r.URL.Path, "/api/sessions/") {
			deleteSessionHandler.ServeHTTP(w, r)
			return
		}
		if r.Method == "DELETE" && strings.HasPrefix(r.URL.Path, "/api/blacklist/") {
			deleteBlacklistHandler.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/verify" {
			rootHandler.ServeHTTP(w, r)
			return
		}
		if r.Method == "GET" {
			fp := filepath.Join(distDir, r.URL.Path)
			if info, err := os.Stat(fp); err == nil && !info.IsDir() {
				fs.ServeHTTP(w, r)
				return
			}
			http.ServeFile(w, r, filepath.Join(distDir, "index.html"))
			return
		}
		rootHandler.ServeHTTP(w, r)
	})

	log.Printf("akguard starting on %s (dev=%v)", addr, devMode)
	if err := http.ListenAndServe(addr, finalHandler); err != nil {
		log.Fatal(err)
	}
}

func onlyMethod(h http.HandlerFunc, methods ...string) http.HandlerFunc {
	allowed := make(map[string]bool)
	for _, m := range methods {
		allowed[m] = true
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if !allowed[r.Method] {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(`{"error":"method not allowed"}`))
			return
		}
		h.ServeHTTP(w, r)
	}
}
