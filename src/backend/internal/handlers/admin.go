package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

type AdminHandler struct {
	db *sql.DB
}

func NewAdminHandler(db *sql.DB) *AdminHandler {
	return &AdminHandler{db: db}
}

func (h *AdminHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/admin/stats", h.stats)
	mux.HandleFunc("GET /api/admin/events/{id}/analytics", h.eventAnalytics)
}

func (h *AdminHandler) stats(w http.ResponseWriter, r *http.Request) {
	var stats struct {
		TotalUsers    int `json:"total_users"`
		TotalEvents   int `json:"total_events"`
		TotalTickets  int `json:"total_tickets"`
		TotalCheckins int `json:"total_checkins"`
	}

	h.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&stats.TotalUsers)
	h.db.QueryRow(`SELECT COUNT(*) FROM events`).Scan(&stats.TotalEvents)
	h.db.QueryRow(`SELECT COUNT(*) FROM tickets`).Scan(&stats.TotalTickets)
	h.db.QueryRow(`SELECT COUNT(*) FROM check_in_log`).Scan(&stats.TotalCheckins)

	writeJSON(w, http.StatusOK, stats)
}

func (h *AdminHandler) eventAnalytics(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("id")

	row := h.db.QueryRow(`
		SELECT total_tickets, total_transfers, transfer_ratio, checked_in_count
		FROM ticket_transfer_stats WHERE event_id = $1`, eventID)

	var total, transfers, checked int
	var ratio float64
	if err := row.Scan(&total, &transfers, &ratio, &checked); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "event not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"event_id":        eventID,
		"total_tickets":   total,
		"total_transfers": transfers,
		"transfer_ratio":  ratio,
		"checked_in":      checked,
	})
}
