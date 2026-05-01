package handler

import (
	"net/http"
	"strings"

	"akguard/internal/config"
	"akguard/internal/middleware"
	"akguard/internal/model"
)

// GET /api/sessions
func GetSessions(state *config.AppState) http.HandlerFunc {
	return middleware.RequireAdminFunc(state, func(w http.ResponseWriter, r *http.Request, entry *config.SessionEntry) {
		sessions := state.Sessions.List()
		middleware.JSONSuccess(w, map[string]interface{}{
			"sessions": sessions,
		})
	})
}

// DELETE /api/sessions/{id}
func DeleteSession(state *config.AppState) http.HandlerFunc {
	return middleware.RequireAdminFunc(state, func(w http.ResponseWriter, r *http.Request, entry *config.SessionEntry) {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 4 {
			middleware.JSONError(w, http.StatusBadRequest, "missing session id")
			return
		}
		id := parts[3]

		state.Sessions.Delete(id)
		model.InsertAuditLog("delete_session", map[string]string{"session_id": id}, resolveIP(r))
		middleware.JSONSuccess(w, map[string]string{"message": "session deleted"})
	})
}
