package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

type TicketHandler struct {
	db *sql.DB
}

func NewTicketHandler(db *sql.DB) *TicketHandler {
	return &TicketHandler{db: db}
}

func (h *TicketHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/events/{eventId}/tickets", h.issue)
	mux.HandleFunc("GET /api/tickets/{id}", h.get)
	mux.HandleFunc("GET /api/my/tickets", h.myTickets)
	mux.HandleFunc("POST /api/tickets/{id}/transfer", h.transfer)
}

type Ticket struct {
	ID         string     `json:"id"`
	EventID    string     `json:"event_id"`
	OwnerID    string     `json:"owner_id"`
	Type       string     `json:"ticket_type"`
	Status     string     `json:"status"`
	QRSecret   string     `json:"qr_secret,omitempty"`
	IssuedAt   time.Time  `json:"issued_at"`
	CheckedIn  *time.Time `json:"checked_in_at,omitempty"`
}

func (h *TicketHandler) issue(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("eventId")
	userID := r.Header.Get("X-User-ID")

	// Submit ZK proof commitment
	var body struct {
		ZKProofCommitment string `json:"zk_proof_commitment"`
		NullifierHash     string `json:"nullifier_hash,omitempty"`
		TicketType        string `json:"ticket_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	// Verify ZK age proof
	if ok, err := verifyZKProof(body.ZKProofCommitment); err != nil || !ok {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "age verification failed"})
		return
	}

	var t Ticket
	err := h.db.QueryRow(
		`INSERT INTO tickets (event_id, owner_id, ticket_type, zk_proof_commitment, nullifier_hash)
		VALUES ($1,$2,$3,$4,$5) RETURNING id, event_id, owner_id, ticket_type, status, qr_secret, issued_at`,
		eventID, userID, body.TicketType, body.ZKProofCommitment, nullIf(body.NullifierHash),
	).Scan(&t.ID, &t.EventID, &t.OwnerID, &t.Type, &t.Status, &t.QRSecret, &t.IssuedAt)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "ticket already exists or event full"})
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (h *TicketHandler) get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var t Ticket
	err := h.db.QueryRow(
		`SELECT id, event_id, owner_id, ticket_type, status, qr_secret, issued_at, checked_in_at
		FROM tickets WHERE id = $1`, id,
	).Scan(&t.ID, &t.EventID, &t.OwnerID, &t.Type, &t.Status, &t.QRSecret, &t.IssuedAt, &t.CheckedIn)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "ticket not found"})
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *TicketHandler) myTickets(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	rows, err := h.db.Query(
		`SELECT t.id, t.event_id, t.owner_id, t.ticket_type, t.status, t.qr_secret, t.issued_at, t.checked_in_at
		FROM tickets t WHERE t.owner_id = $1 ORDER BY t.issued_at DESC`, userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	var tickets []Ticket
	for rows.Next() {
		var t Ticket
		if err := rows.Scan(&t.ID, &t.EventID, &t.OwnerID, &t.Type, &t.Status, &t.QRSecret, &t.IssuedAt, &t.CheckedIn); err != nil {
			continue
		}
		t.QRSecret = ""
		tickets = append(tickets, t)
	}
	if tickets == nil {
		tickets = []Ticket{}
	}
	writeJSON(w, http.StatusOK, tickets)
}

func (h *TicketHandler) transfer(w http.ResponseWriter, r *http.Request) {
	ticketID := r.PathValue("id")
	fromUserID := r.Header.Get("X-User-ID")

	var body struct {
		ToUserID    string `json:"to_user_id"`
		ZKProofFrom string `json:"zk_proof_from"`
		ZKProofTo   string `json:"zk_proof_to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	// Log transfer
	_, err := h.db.Exec(
		`INSERT INTO transfer_log (ticket_id, from_user_id, to_user_id, from_proof_commitment, to_proof_commitment)
		VALUES ($1,$2,$3,$4,$5)`, ticketID, fromUserID, body.ToUserID, body.ZKProofFrom, body.ZKProofTo)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Update ticket ownership
	_, err = h.db.Exec(`UPDATE tickets SET owner_id = $1, status = 'transferred' WHERE id = $2 AND owner_id = $3`,
		body.ToUserID, ticketID, fromUserID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "transfer failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "transferred"})
}
