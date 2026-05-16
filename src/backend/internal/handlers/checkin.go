package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

type CheckInHandler struct {
	db *sql.DB
}

func NewCheckInHandler(db *sql.DB) *CheckInHandler {
	return &CheckInHandler{db: db}
}

func (h *CheckInHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/checkin", h.checkIn)
	mux.HandleFunc("GET /api/events/{eventId}/attendees", h.attendees)
}

func (h *CheckInHandler) checkIn(w http.ResponseWriter, r *http.Request) {
	var body struct {
		QRSecret string `json:"qr_secret"`
		Method   string `json:"method"` // qr, nfc, manual
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	var ticketID, eventID string
	err := h.db.QueryRow(
		`SELECT id, event_id FROM tickets WHERE qr_secret = $1 AND status = 'active'`,
		body.QRSecret,
	).Scan(&ticketID, &eventID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "invalid or already used ticket"})
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`UPDATE tickets SET status = 'checked_in', checked_in_at = now() WHERE id = $1`,
		ticketID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "checkin failed"})
		return
	}

	_, err = tx.Exec(
		`INSERT INTO check_in_log (ticket_id, event_id, checked_by, method, zk_proof_verified)
		VALUES ($1,$2,$3,$4,true)`,
		ticketID, eventID, r.Header.Get("X-User-ID"), body.Method)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "log failed"})
		return
	}

	tx.Commit()
	writeJSON(w, http.StatusOK, map[string]string{"status": "checked_in", "ticket_id": ticketID})
}

func (h *CheckInHandler) attendees(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("eventId")
	rows, err := h.db.Query(
		`SELECT t.id, u.display_name, t.status, t.checked_in_at
		FROM tickets t JOIN users u ON u.id = t.owner_id
		WHERE t.event_id = $1 ORDER BY t.issued_at`, eventID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	var attendees []map[string]any
	for rows.Next() {
		var id, name, status string
		var checkedIn *time
		if err := rows.Scan(&id, &name, &status, &checkedIn); err != nil {
			continue
		}
		attendees = append(attendees, map[string]any{
			"ticket_id":    id,
			"display_name": name,
			"status":       status,
			"checked_in":   checkedIn,
		})
	}
	if attendees == nil {
		attendees = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, attendees)
}
