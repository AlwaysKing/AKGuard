package handler

import (
	"net/http"
	"strconv"

	"akguard/internal/config"
	"akguard/internal/middleware"
	"akguard/internal/model"
)

// GET /api/logs/access
func GetAccessLogs(state *config.AppState) http.HandlerFunc {
	return middleware.RequireAdminFunc(state, func(w http.ResponseWriter, r *http.Request, claims *config.SessionEntry) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
		host := r.URL.Query().Get("host")

		if page < 1 {
			page = 1
		}
		if pageSize < 1 || pageSize > 100 {
			pageSize = 20
		}

		result, err := model.QueryAccessLogs(page, pageSize, host)
		if err != nil {
			middleware.JSONError(w, http.StatusInternalServerError, "failed to query logs")
			return
		}

		middleware.JSONSuccess(w, result)
	})
}

// GET /api/logs/access/stats
func GetAccessLogStats(state *config.AppState) http.HandlerFunc {
	return middleware.RequireAdminFunc(state, func(w http.ResponseWriter, r *http.Request, claims *config.SessionEntry) {
		stats, err := model.GetAccessLogStats()
		if err != nil {
			middleware.JSONError(w, http.StatusInternalServerError, "failed to query stats")
			return
		}
		middleware.JSONSuccess(w, stats)
	})
}

// GET /api/logs/audit
func GetAuditLogs(state *config.AppState) http.HandlerFunc {
	return middleware.RequireAdminFunc(state, func(w http.ResponseWriter, r *http.Request, claims *config.SessionEntry) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

		if page < 1 {
			page = 1
		}
		if pageSize < 1 || pageSize > 100 {
			pageSize = 20
		}

		result, err := model.QueryAuditLogs(page, pageSize)
		if err != nil {
			middleware.JSONError(w, http.StatusInternalServerError, "failed to query audit logs")
			return
		}

		middleware.JSONSuccess(w, result)
	})
}
