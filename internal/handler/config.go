package handler

import (
	"encoding/json"
	"net"
	"net/http"

	"akguard/internal/config"
	"akguard/internal/middleware"
	"akguard/internal/model"

	"strconv"

	"golang.org/x/crypto/bcrypt"
)

// GET /api/config
func GetConfig(state *config.AppState) http.HandlerFunc {
	return middleware.RequireAdminFunc(state, func(w http.ResponseWriter, r *http.Request, entry *config.SessionEntry) {
		cfg := state.Config
		isDefaultAuth := bcrypt.CompareHashAndPassword(
			[]byte(cfg.Auth.AuthPassword),
			[]byte(config.DefaultPasswd),
		) == nil

		middleware.JSONSuccess(w, map[string]interface{}{
			"internal_nets":        cfg.Auth.InternalNets,
			"default_policy":       cfg.Auth.DefaultPolicy,
			"domains":              cfg.Auth.Domains,
			"bark_url":             cfg.Auth.BarkURL,
			"has_bark":             cfg.Auth.BarkURL != "",
			"is_default_auth":      isDefaultAuth,
			"site_title":           cfg.Auth.SiteTitle,
			"admin_password_login": cfg.Auth.AdminPasswordLogin,
			"admin_bark_login":     cfg.Auth.AdminBarkLogin,
			"auth_password_login":  cfg.Auth.AuthPasswordLogin,
			"auth_bark_login":      cfg.Auth.AuthBarkLogin,
			"auth_apikey_login":    cfg.Auth.AuthApiKeyLogin,
			"token_grace_period":   cfg.Auth.TokenGracePeriod,
		})
	})
}

// GET /api/login-methods — 公开端点，返回各类登录方式是否可用
func GetLoginMethods(state *config.AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := state.Config
		middleware.JSONSuccess(w, map[string]interface{}{
			"site_title":           cfg.Auth.SiteTitle,
			"admin_password_login": cfg.Auth.AdminPasswordLogin,
			"admin_bark_login":     cfg.Auth.AdminBarkLogin,
			"auth_password_login":  cfg.Auth.AuthPasswordLogin,
			"auth_bark_login":      cfg.Auth.AuthBarkLogin,
			"auth_apikey_login":    cfg.Auth.AuthApiKeyLogin,
			"token_grace_period":   cfg.Auth.TokenGracePeriod,
		})
	}
}

// PUT /api/config/auth-password
func UpdateAuthPassword(state *config.AppState) http.HandlerFunc {
	return middleware.RequireAdminFunc(state, func(w http.ResponseWriter, r *http.Request, entry *config.SessionEntry) {
		var req struct {
			NewPassword     string `json:"new_password"`
			ConfirmPassword string `json:"confirm_password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			middleware.JSONError(w, http.StatusBadRequest, "invalid request")
			return
		}
		if req.NewPassword != req.ConfirmPassword {
			middleware.JSONError(w, http.StatusBadRequest, "passwords do not match")
			return
		}

		hashed := config.HashPassword(req.NewPassword)
		state.Config.Auth.AuthPassword = hashed
		model.ConfigSet("auth_password", hashed)

		model.InsertAuditLog("change_auth_password", nil, resolveIP(r))
		middleware.JSONSuccess(w, map[string]string{"message": "auth password updated"})
	})
}

// PUT /api/config/admin-password
func UpdateAdminPassword(state *config.AppState) http.HandlerFunc {
	return middleware.RequireAdminFunc(state, func(w http.ResponseWriter, r *http.Request, entry *config.SessionEntry) {
		var req struct {
			CurrentPassword string `json:"current_password"`
			NewPassword     string `json:"new_password"`
			ConfirmPassword string `json:"confirm_password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			middleware.JSONError(w, http.StatusBadRequest, "invalid request")
			return
		}

		if err := bcrypt.CompareHashAndPassword(
			[]byte(state.Config.Auth.AdminPassword),
			[]byte(req.CurrentPassword),
		); err != nil {
			middleware.JSONError(w, http.StatusUnauthorized, "current password incorrect")
			return
		}

		if req.NewPassword != req.ConfirmPassword {
			middleware.JSONError(w, http.StatusBadRequest, "passwords do not match")
			return
		}

		hashed := config.HashPassword(req.NewPassword)
		state.Config.Auth.AdminPassword = hashed
		model.ConfigSet("admin_password", hashed)

		// 删掉旧的 admin session，创建新的
		state.Sessions.Delete(entry.ID)
		newSessionID := state.Sessions.Create(resolveIP(r), r.UserAgent(), "admin", false)
		middleware.SetCookie(w, r, middleware.AdminCookieName, newSessionID)

		model.InsertAuditLog("change_admin_password", nil, resolveIP(r))
		middleware.JSONSuccess(w, map[string]string{"message": "admin password updated"})
	})
}

// PUT /api/config/bark
func UpdateBark(state *config.AppState) http.HandlerFunc {
	return middleware.RequireAdminFunc(state, func(w http.ResponseWriter, r *http.Request, entry *config.SessionEntry) {
		var req struct {
			BarkURL string `json:"bark_url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			middleware.JSONError(w, http.StatusBadRequest, "invalid request")
			return
		}

		state.Config.Auth.BarkURL = req.BarkURL
		model.ConfigSet("bark_url", req.BarkURL)

		model.InsertAuditLog("update_bark", map[string]string{"bark_url": req.BarkURL}, resolveIP(r))
		middleware.JSONSuccess(w, map[string]string{"message": "bark url updated"})
	})
}

// POST /api/config/test-bark — 发送验证码到指定 Bark 地址
func TestBark(state *config.AppState) http.HandlerFunc {
	return middleware.RequireAdminFunc(state, func(w http.ResponseWriter, r *http.Request, entry *config.SessionEntry) {
		var req struct {
			BarkURL string `json:"bark_url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			middleware.JSONError(w, http.StatusBadRequest, "invalid request")
			return
		}
		if req.BarkURL == "" {
			middleware.JSONError(w, http.StatusBadRequest, "bark url is required")
			return
		}

		code := middleware.GenerateBarkVerifyCode()
		if err := middleware.SendBarkPush(req.BarkURL, "验证码: "+code); err != nil {
			middleware.JSONError(w, http.StatusBadGateway, "push failed: "+err.Error())
			return
		}

		middleware.JSONSuccess(w, map[string]string{"message": "verification code sent"})
	})
}

// POST /api/config/confirm-bark — 验证码校验通过后保存 Bark 地址
func ConfirmBark(state *config.AppState) http.HandlerFunc {
	return middleware.RequireAdminFunc(state, func(w http.ResponseWriter, r *http.Request, entry *config.SessionEntry) {
		var req struct {
			BarkURL string `json:"bark_url"`
			Code    string `json:"code"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			middleware.JSONError(w, http.StatusBadRequest, "invalid request")
			return
		}

		if !middleware.VerifyBarkCode(req.Code) {
			middleware.JSONError(w, http.StatusBadRequest, "invalid verification code")
			return
		}

		state.Config.Auth.BarkURL = req.BarkURL
		model.ConfigSet("bark_url", req.BarkURL)

		model.InsertAuditLog("update_bark", map[string]string{"bark_url": req.BarkURL}, resolveIP(r))
		middleware.JSONSuccess(w, map[string]string{"message": "bark url verified and saved"})
	})
}

// PUT /api/config/networks
func UpdateNetworks(state *config.AppState) http.HandlerFunc {
	return middleware.RequireAdminFunc(state, func(w http.ResponseWriter, r *http.Request, entry *config.SessionEntry) {
		var req struct {
			Networks []string `json:"networks"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			middleware.JSONError(w, http.StatusBadRequest, "invalid request")
			return
		}

		for _, cidr := range req.Networks {
			if _, _, err := net.ParseCIDR(cidr); err != nil {
				middleware.JSONError(w, http.StatusBadRequest, "invalid CIDR: "+cidr)
				return
			}
		}

		state.Config.Auth.InternalNets = req.Networks
		state.ParsedNets = config.ParseCIDRs(req.Networks)
		model.SetInternalNets(req.Networks)

		model.InsertAuditLog("update_networks", map[string][]string{"networks": req.Networks}, resolveIP(r))
		middleware.JSONSuccess(w, map[string]string{"message": "networks updated"})
	})
}

// PUT /api/config/domains
func UpdateDomains(state *config.AppState) http.HandlerFunc {
	return middleware.RequireAdminFunc(state, func(w http.ResponseWriter, r *http.Request, entry *config.SessionEntry) {
		var req struct {
			Domains []config.DomainRule `json:"domains"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			middleware.JSONError(w, http.StatusBadRequest, "invalid request")
			return
		}

		for _, d := range req.Domains {
			if !config.IsValidAction(d.Internal) || !config.IsValidAction(d.External) {
				middleware.JSONError(w, http.StatusBadRequest, "invalid policy for domain: "+d.Host)
				return
			}
		}

		state.Config.Auth.Domains = req.Domains
		dbRules := make([]model.DomainRuleDB, len(req.Domains))
		for i, d := range req.Domains {
			dbRules[i] = model.DomainRuleDB{
				Host:     d.Host,
				Internal: d.Internal,
				External: d.External,
			}
		}
		model.SetDomainRules(dbRules)

		model.InsertAuditLog("update_domains", map[string][]config.DomainRule{"domains": req.Domains}, resolveIP(r))
		middleware.JSONSuccess(w, map[string]string{"message": "domains updated"})
	})
}

// PUT /api/config/default-policy
func UpdateDefaultPolicy(state *config.AppState) http.HandlerFunc {
	return middleware.RequireAdminFunc(state, func(w http.ResponseWriter, r *http.Request, entry *config.SessionEntry) {
		var req struct {
			Internal string `json:"internal"`
			External string `json:"external"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			middleware.JSONError(w, http.StatusBadRequest, "invalid request")
			return
		}

		if !config.IsValidAction(req.Internal) || !config.IsValidAction(req.External) {
			middleware.JSONError(w, http.StatusBadRequest, "invalid policy value")
			return
		}

		state.Config.Auth.DefaultPolicy = config.Policy{Internal: req.Internal, External: req.External}
		model.ConfigSet("default_internal_policy", req.Internal)
		model.ConfigSet("default_external_policy", req.External)

		model.InsertAuditLog("update_default_policy", req, resolveIP(r))
		middleware.JSONSuccess(w, map[string]string{"message": "default policy updated"})
	})
}

// PUT /api/config/admin-login-methods
func UpdateAdminLoginMethods(state *config.AppState) http.HandlerFunc {
	return middleware.RequireAdminFunc(state, func(w http.ResponseWriter, r *http.Request, entry *config.SessionEntry) {
		var req struct {
			PasswordLogin bool `json:"password_login"`
			BarkLogin     bool `json:"bark_login"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			middleware.JSONError(w, http.StatusBadRequest, "invalid request")
			return
		}
		if !req.PasswordLogin && !req.BarkLogin {
			middleware.JSONError(w, http.StatusBadRequest, "at least one login method must be enabled")
			return
		}
		if req.BarkLogin && state.Config.Auth.BarkURL == "" {
			middleware.JSONError(w, http.StatusBadRequest, "bark must be configured before enabling bark login")
			return
		}

		state.Config.Auth.AdminPasswordLogin = req.PasswordLogin
		state.Config.Auth.AdminBarkLogin = req.BarkLogin
		model.ConfigSet("admin_password_login", boolToStr(req.PasswordLogin))
		model.ConfigSet("admin_bark_login", boolToStr(req.BarkLogin))

		model.InsertAuditLog("update_admin_login_methods", req, resolveIP(r))
		middleware.JSONSuccess(w, map[string]string{"message": "admin login methods updated"})
	})
}

// PUT /api/config/auth-login-methods
func UpdateAuthLoginMethods(state *config.AppState) http.HandlerFunc {
	return middleware.RequireAdminFunc(state, func(w http.ResponseWriter, r *http.Request, entry *config.SessionEntry) {
		var req struct {
			PasswordLogin bool `json:"password_login"`
			BarkLogin     bool `json:"bark_login"`
			ApiKeyLogin   bool `json:"apikey_login"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			middleware.JSONError(w, http.StatusBadRequest, "invalid request")
			return
		}
		if !req.PasswordLogin && !req.BarkLogin && !req.ApiKeyLogin {
			middleware.JSONError(w, http.StatusBadRequest, "at least one login method must be enabled")
			return
		}
		if req.BarkLogin && state.Config.Auth.BarkURL == "" {
			middleware.JSONError(w, http.StatusBadRequest, "bark must be configured before enabling bark login")
			return
		}

		state.Config.Auth.AuthPasswordLogin = req.PasswordLogin
		state.Config.Auth.AuthBarkLogin = req.BarkLogin
		state.Config.Auth.AuthApiKeyLogin = req.ApiKeyLogin
		model.ConfigSet("auth_password_login", boolToStr(req.PasswordLogin))
		model.ConfigSet("auth_bark_login", boolToStr(req.BarkLogin))
		model.ConfigSet("auth_apikey_login", boolToStr(req.ApiKeyLogin))

		model.InsertAuditLog("update_auth_login_methods", req, resolveIP(r))
		middleware.JSONSuccess(w, map[string]string{"message": "auth login methods updated"})
	})
}

func boolToStr(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// PUT /api/config/site-title
func UpdateSiteTitle(state *config.AppState) http.HandlerFunc {
	return middleware.RequireAdminFunc(state, func(w http.ResponseWriter, r *http.Request, entry *config.SessionEntry) {
		var req struct {
			Title string `json:"title"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			middleware.JSONError(w, http.StatusBadRequest, "invalid request")
			return
		}
		if req.Title == "" {
			middleware.JSONError(w, http.StatusBadRequest, "title cannot be empty")
			return
		}

		state.Config.Auth.SiteTitle = req.Title
		model.ConfigSet("site_title", req.Title)

		model.InsertAuditLog("update_site_title", map[string]string{"title": req.Title}, resolveIP(r))
		middleware.JSONSuccess(w, map[string]string{"message": "site title updated"})
	})
}

// PUT /api/config/token-grace-period
func UpdateTokenGracePeriod(state *config.AppState) http.HandlerFunc {
	return middleware.RequireAdminFunc(state, func(w http.ResponseWriter, r *http.Request, entry *config.SessionEntry) {
		var req struct {
			Seconds int `json:"seconds"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			middleware.JSONError(w, http.StatusBadRequest, "invalid request")
			return
		}
		if req.Seconds < 0 {
			middleware.JSONError(w, http.StatusBadRequest, "seconds must be >= 0")
			return
		}

		state.Config.Auth.TokenGracePeriod = req.Seconds
		model.ConfigSet("token_grace_period", strconv.Itoa(req.Seconds))

		model.InsertAuditLog("update_token_grace_period", req, resolveIP(r))
		middleware.JSONSuccess(w, map[string]string{"message": "token grace period updated"})
	})
}

// POST /api/config/regenerate-apikey — 重新生成 API Key
func RegenerateApiKey(state *config.AppState) http.HandlerFunc {
	return middleware.RequireAdminFunc(state, func(w http.ResponseWriter, r *http.Request, entry *config.SessionEntry) {
		newKey := config.GenerateApiKey()
		state.Config.Auth.ApiKey = newKey
		model.ConfigSet("api_key", newKey)

		model.InsertAuditLog("regenerate_apikey", nil, resolveIP(r))
		middleware.JSONSuccess(w, map[string]string{"api_key": newKey})
	})
}
