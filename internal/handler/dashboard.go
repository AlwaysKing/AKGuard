package handler

import (
	"net/http"

	"akguard/internal/config"
	"akguard/internal/middleware"
	"akguard/internal/model"
)

// GET /api/dashboard
func GetDashboard(state *config.AppState) http.HandlerFunc {
	return middleware.RequireAdminFunc(state, func(w http.ResponseWriter, r *http.Request, entry *config.SessionEntry) {
		stats, err := model.GetAccessLogStats()
		if err != nil {
			stats = &model.AccessLogStats{}
		}

		activeSessions := state.Sessions.Count()
		recentLogs, _ := model.QueryAccessLogs(1, 10, "")

		middleware.JSONSuccess(w, map[string]interface{}{
			"stats":           stats,
			"active_sessions": activeSessions,
			"recent_logs":     recentLogs.Logs,
			"domains_count":   len(state.Config.Auth.Domains),
		})
	})
}
