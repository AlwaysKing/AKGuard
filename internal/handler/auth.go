package handler

import (
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"akguard/internal/config"
	"akguard/internal/middleware"
	"akguard/internal/model"

	"golang.org/x/crypto/bcrypt"
)

// resolveIP 从请求中获取客户端 IP
func resolveIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}

func shortID(s string) string {
	if len(s) > 12 {
		return s[:12]
	}
	return s
}

// checkBan 检查 IP 是否被封禁，返回 true 表示已被封禁
func checkBan(state *config.AppState, ip, banType string) bool {
	return model.IsBlacklisted(ip, banType)
}

// recordFailure 记录登录失败，达到阈值自动封禁
func recordFailure(state *config.AppState, ip, banType string) {
	bc := getBanConfig(state, banType)
	if !bc.Enabled {
		return
	}
	count := state.Attempts.Record(banType, ip, bc.WindowSec)
	if count >= bc.MaxAttempts {
		_ = model.AddToBlacklist(ip, banType, "auto", time.Duration(bc.DurationSec)*time.Second)
		log.Printf("[ban] auto-ban type=%s ip=%s for %ds (attempts=%d)", banType, ip, bc.DurationSec, count)
		state.Attempts.Clear(banType, ip)
	}
}

func getBanConfig(state *config.AppState, banType string) config.BanConfig {
	if banType == "admin" {
		return state.Config.Auth.AdminBan
	}
	return state.Config.Auth.AuthBan
}

// POST /api/auth/login
func AuthLogin(state *config.AppState, otpStore *middleware.OTPStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := resolveIP(r)

		// 检查是否被封禁
		if checkBan(state, ip, "auth") {
			middleware.JSONError(w, http.StatusForbidden, "ip is banned")
			return
		}

		var req struct {
			Password string `json:"password"`
			OTPCode  string `json:"otp_code"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			middleware.JSONError(w, http.StatusBadRequest, "invalid request")
			return
		}

		isInit := false
		failed := false

		// OTP 验证
		if req.OTPCode != "" {
			if !state.Config.Auth.AuthBarkLogin {
				middleware.JSONError(w, http.StatusBadRequest, "bark login is not enabled")
				return
			}
			if !otpStore.Verify("auth", req.OTPCode) {
				failed = true
			}
		} else {
			// 密码验证
			if !state.Config.Auth.AuthPasswordLogin {
				middleware.JSONError(w, http.StatusBadRequest, "password login is not enabled")
				return
			}
			if err := bcrypt.CompareHashAndPassword(
				[]byte(state.Config.Auth.AuthPassword),
				[]byte(req.Password),
			); err != nil {
				failed = true
			} else {
				isInit = (req.Password == config.DefaultPasswd)
			}
		}

		if failed {
			recordFailure(state, ip, "auth")
			middleware.JSONError(w, http.StatusUnauthorized, "invalid password or code")
			return
		}

		// 登录成功，清除失败计数
		state.Attempts.Clear("auth", ip)

		sessionID := state.Sessions.Create(ip, r.UserAgent(), "auth", isInit)
		log.Printf("[auth-login] session=%s ip=%s", shortID(sessionID), ip)
		middleware.SetCookie(w, r, middleware.AuthCookieName, sessionID)
		middleware.JSONSuccess(w, map[string]interface{}{
			"ok":                  true,
			"is_default_password": isInit,
		})
	}
}

// POST /api/auth/renew — token 自动续签
func AuthRenew(state *config.AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		gracePeriod := state.Config.Auth.TokenGracePeriod
		if gracePeriod <= 0 {
			middleware.JSONError(w, http.StatusUnauthorized, "renew not enabled")
			return
		}

		cookie, err := r.Cookie(middleware.AuthCookieName)
		if err != nil {
			middleware.JSONError(w, http.StatusUnauthorized, "no token")
			return
		}

		ip := resolveIP(r)
		entry := state.Sessions.GetAny(cookie.Value)
		if entry == nil {
			middleware.JSONError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		if !config.IpEqual(entry.ClientIP, ip) {
			middleware.JSONError(w, http.StatusUnauthorized, "ip mismatch")
			return
		}

		// 检查宽限期：expires_at + grace_period > now
		deadline := entry.ExpiresAt.Add(time.Duration(gracePeriod) * time.Second)
		if time.Now().After(deadline) {
			middleware.JSONError(w, http.StatusUnauthorized, "token expired beyond grace period")
			return
		}

		// 删除旧 session，创建新 session
		state.Sessions.Delete(cookie.Value)
		newID := state.Sessions.Create(ip, r.UserAgent(), "auth", entry.IsInit)
		log.Printf("[auth-renew] old=%s new=%s ip=%s", shortID(cookie.Value), shortID(newID), ip)
		middleware.SetCookie(w, r, middleware.AuthCookieName, newID)
		middleware.JSONSuccess(w, map[string]bool{"ok": true})
	}
}

// POST /api/auth/logout
func AuthLogout(state *config.AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie(middleware.AuthCookieName); err == nil {
			state.Sessions.Delete(cookie.Value)
		}
		middleware.ClearCookie(w, r, middleware.AuthCookieName)
		middleware.JSONSuccess(w, map[string]bool{"ok": true})
	}
}

// POST /api/admin/login
func AdminLogin(state *config.AppState, otpStore *middleware.OTPStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := resolveIP(r)

		// 检查是否被封禁
		if checkBan(state, ip, "admin") {
			middleware.JSONError(w, http.StatusForbidden, "ip is banned")
			return
		}

		var req struct {
			Password string `json:"password"`
			OTPCode  string `json:"otp_code"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			middleware.JSONError(w, http.StatusBadRequest, "invalid request")
			return
		}

		failed := false

		// OTP 验证
		if req.OTPCode != "" {
			if !state.Config.Auth.AdminBarkLogin {
				middleware.JSONError(w, http.StatusBadRequest, "bark login is not enabled")
				return
			}
			if !otpStore.Verify("admin", req.OTPCode) {
				failed = true
			}
		} else {
			// 密码验证
			if !state.Config.Auth.AdminPasswordLogin {
				middleware.JSONError(w, http.StatusBadRequest, "password login is not enabled")
				return
			}
			if err := bcrypt.CompareHashAndPassword(
				[]byte(state.Config.Auth.AdminPassword),
				[]byte(req.Password),
			); err != nil {
				failed = true
			}
		}

		if failed {
			recordFailure(state, ip, "admin")
			middleware.JSONError(w, http.StatusUnauthorized, "invalid password or code")
			return
		}

		// 登录成功，清除失败计数
		state.Attempts.Clear("admin", ip)

		sessionID := state.Sessions.Create(ip, r.UserAgent(), "admin", false)
		middleware.SetCookie(w, r, middleware.AdminCookieName, sessionID)
		middleware.JSONSuccess(w, map[string]bool{"ok": true})
	}
}

// POST /api/admin/logout
func AdminLogout(state *config.AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie(middleware.AdminCookieName); err == nil {
			state.Sessions.Delete(cookie.Value)
		}
		middleware.ClearCookie(w, r, middleware.AdminCookieName)
		middleware.JSONSuccess(w, map[string]bool{"ok": true})
	}
}

// POST /api/otp/send
func SendOTP(state *config.AppState, otpStore *middleware.OTPStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Type string `json:"type"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		otpType := req.Type
		if otpType != "auth" && otpType != "admin" {
			otpType = "auth"
		}

		if otpType == "auth" && !state.Config.Auth.AuthBarkLogin {
			middleware.JSONError(w, http.StatusBadRequest, "bark login is not enabled for auth")
			return
		}
		if otpType == "admin" && !state.Config.Auth.AdminBarkLogin {
			middleware.JSONError(w, http.StatusBadRequest, "bark login is not enabled for admin")
			return
		}

		if state.Config.Auth.BarkURL == "" {
			middleware.JSONError(w, http.StatusBadRequest, "bark not configured")
			return
		}

		if cooledDown, retryAfter := otpStore.CheckCooldown(otpType); cooledDown {
			middleware.JSONResponse(w, http.StatusTooManyRequests, map[string]interface{}{
				"error":       "cooldown",
				"retry_after": retryAfter,
			})
			return
		}

		code := otpStore.Generate()
		otpStore.Set(otpType, code)

		if err := middleware.SendBarkPush(state.Config.Auth.BarkURL, code); err != nil {
			middleware.JSONError(w, http.StatusBadGateway, "push failed")
			return
		}

		middleware.JSONSuccess(w, map[string]bool{"ok": true})
	}
}

// GET /verify — nginx auth_request 端点
func Verify(state *config.AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		policy := config.ResolvePolicy(host, state.Config)

		realIP := r.Header.Get("X-Real-IP")
		internal := config.IsInternalIP(realIP, state.ParsedNets)

		var action string
		if internal {
			action = policy.Internal
		} else {
			action = policy.External
		}

		// 记录访问日志（异步）
		go func() {
			result := "denied"
			if action == "pass" || action == "auth" {
				result = "allowed"
			}
			source := "external"
			if internal {
				source = "internal"
			}
			if err := model.InsertAccessLog(&model.AccessLog{
				Host:      host,
				ClientIP:  realIP,
				Source:    source,
				Action:    action,
				Result:    result,
				UserAgent: r.UserAgent(),
			}); err != nil {
				log.Printf("failed to insert access log: %v", err)
			}
		}()

		switch action {
		case "reject":
			w.WriteHeader(http.StatusUnauthorized)
		case "pass":
			w.WriteHeader(http.StatusOK)
		case "auth":
			// 按来源分流：cookie → session 验证，header/URL → API Key 验证
			if cookie, err := r.Cookie(middleware.AuthCookieName); err == nil {
				// Cookie 来源 → Session 路径
				clientIP := realIP
				if clientIP == "" {
					clientIP = r.RemoteAddr
				}
				entry, ok := state.Sessions.Validate(cookie.Value, clientIP, "auth")
				if !ok || entry.IsInit {
					log.Printf("[verify] host=%s action=auth invalid session=%s... clientIP=%s remote=%s ok=%v isInit=%v entry_nil=%v",
						host, shortID(cookie.Value), clientIP, r.RemoteAddr, ok, entry != nil && entry.IsInit, entry == nil)
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				log.Printf("[verify] host=%s action=auth OK session=%s...", host, shortID(cookie.Value))
				w.WriteHeader(http.StatusOK)
				return
			}

			// Header / URL 参数来源 → API Key 路径
			var apiKey string
			if h := r.Header.Get("X-AK-Token"); h != "" {
				apiKey = h
			} else if q := r.URL.Query().Get("ak_token"); q != "" {
				apiKey = q
			}

			if apiKey == "" {
				log.Printf("[verify] host=%s action=auth no_token remote=%s", host, r.RemoteAddr)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			if !state.Config.Auth.AuthApiKeyLogin {
				log.Printf("[verify] host=%s apikey auth disabled remote=%s", host, r.RemoteAddr)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			if subtle.ConstantTimeCompare([]byte(apiKey), []byte(state.Config.Auth.ApiKey)) != 1 {
				log.Printf("[verify] host=%s apikey mismatch remote=%s", host, r.RemoteAddr)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			log.Printf("[verify] host=%s action=auth OK via apikey remote=%s", host, r.RemoteAddr)
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusUnauthorized)
		}
	}
}

