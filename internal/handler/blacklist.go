package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"akguard/internal/config"
	"akguard/internal/middleware"
	"akguard/internal/model"
)

// GET /api/blacklist/:type
func GetBlacklist(state *config.AppState) http.HandlerFunc {
	return middleware.RequireAdminFunc(state, func(w http.ResponseWriter, r *http.Request, entry *config.SessionEntry) {
		banType := r.URL.Query().Get("type")
		if banType != "auth" && banType != "admin" {
			banType = "auth"
		}

		model.CleanupExpiredBlacklist()
		entries, err := model.GetBlacklist(banType)
		if err != nil {
			middleware.JSONError(w, http.StatusInternalServerError, "failed to load blacklist")
			return
		}
		if entries == nil {
			entries = []model.BlacklistEntry{}
		}
		middleware.JSONSuccess(w, map[string]interface{}{
			"entries": entries,
		})
	})
}

// POST /api/blacklist 手动添加
func AddBlacklist(state *config.AppState) http.HandlerFunc {
	return middleware.RequireAdminFunc(state, func(w http.ResponseWriter, r *http.Request, entry *config.SessionEntry) {
		var req struct {
			IP        string `json:"ip"`
			Type      string `json:"type"`
			Duration  int    `json:"duration_sec"` // 0 = 永久
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			middleware.JSONError(w, http.StatusBadRequest, "invalid request")
			return
		}
		if req.IP == "" {
			middleware.JSONError(w, http.StatusBadRequest, "ip is required")
			return
		}
		if req.Type != "auth" && req.Type != "admin" {
			middleware.JSONError(w, http.StatusBadRequest, "type must be auth or admin")
			return
		}

		var dur time.Duration
		if req.Duration > 0 {
			dur = time.Duration(req.Duration) * time.Second
		}

		if err := model.AddToBlacklist(req.IP, req.Type, "manual", dur); err != nil {
			middleware.JSONError(w, http.StatusInternalServerError, "failed to add")
			return
		}

		model.InsertAuditLog("add_blacklist", map[string]string{"ip": req.IP, "type": req.Type}, resolveIP(r))
		middleware.JSONSuccess(w, map[string]string{"message": "added to blacklist"})
	})
}

// DELETE /api/blacklist/:id
func DeleteBlacklist(state *config.AppState) http.HandlerFunc {
	return middleware.RequireAdminFunc(state, func(w http.ResponseWriter, r *http.Request, entry *config.SessionEntry) {
		parts := splitPath(r.URL.Path)
		if len(parts) < 3 {
			middleware.JSONError(w, http.StatusBadRequest, "missing id")
			return
		}
		id, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			middleware.JSONError(w, http.StatusBadRequest, "invalid id")
			return
		}

		if err := model.RemoveFromBlacklist(id); err != nil {
			middleware.JSONError(w, http.StatusInternalServerError, "failed to remove")
			return
		}

		model.InsertAuditLog("remove_blacklist", map[string]string{"id": strconv.FormatInt(id, 10)}, resolveIP(r))
		middleware.JSONSuccess(w, map[string]string{"message": "removed from blacklist"})
	})
}

// PUT /api/config/auth-ban
func UpdateAuthBanConfig(state *config.AppState) http.HandlerFunc {
	return middleware.RequireAdminFunc(state, func(w http.ResponseWriter, r *http.Request, entry *config.SessionEntry) {
		var req config.BanConfig
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			middleware.JSONError(w, http.StatusBadRequest, "invalid request")
			return
		}

		state.Config.Auth.AuthBan = req
		model.ConfigSet("auth_ban_enabled", boolToStr(req.Enabled))
		model.ConfigSet("auth_ban_window", strconv.Itoa(req.WindowSec))
		model.ConfigSet("auth_ban_max_attempts", strconv.Itoa(req.MaxAttempts))
		model.ConfigSet("auth_ban_duration", strconv.Itoa(req.DurationSec))

		model.InsertAuditLog("update_auth_ban_config", req, resolveIP(r))
		middleware.JSONSuccess(w, map[string]string{"message": "auth ban config updated"})
	})
}

// PUT /api/config/admin-ban
func UpdateAdminBanConfig(state *config.AppState) http.HandlerFunc {
	return middleware.RequireAdminFunc(state, func(w http.ResponseWriter, r *http.Request, entry *config.SessionEntry) {
		var req config.BanConfig
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			middleware.JSONError(w, http.StatusBadRequest, "invalid request")
			return
		}

		state.Config.Auth.AdminBan = req
		model.ConfigSet("admin_ban_enabled", boolToStr(req.Enabled))
		model.ConfigSet("admin_ban_window", strconv.Itoa(req.WindowSec))
		model.ConfigSet("admin_ban_max_attempts", strconv.Itoa(req.MaxAttempts))
		model.ConfigSet("admin_ban_duration", strconv.Itoa(req.DurationSec))

		model.InsertAuditLog("update_admin_ban_config", req, resolveIP(r))
		middleware.JSONSuccess(w, map[string]string{"message": "admin ban config updated"})
	})
}

func splitPath(p string) []string {
	var parts []string
	for _, s := range splitAfter(p, '/') {
		if s != "" && s != "/" {
			parts = append(parts, s)
		}
	}
	return parts
}

func splitAfter(s string, sep byte) []string {
	var result []string
	i := 0
	for j := 0; j < len(s); j++ {
		if s[j] == sep {
			result = append(result, s[i:j+1])
			i = j + 1
		}
	}
	if i < len(s) {
		result = append(result, s[i:])
	}
	return result
}
