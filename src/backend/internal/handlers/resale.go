// TODO: Register this handler in cmd/server/main.go
package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

type ResaleHandler struct {
	db *sql.DB
}

func NewResaleHandler(db *sql.DB) *ResaleHandler {
	return &ResaleHandler{db: db}
}

func (h *ResaleHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/tickets/{id}/list", h.listTicket)
	mux.HandleFunc("GET /api/events/{id}/resale", h.browseListings)
	mux.HandleFunc("POST /api/resale/{id}/buy", h.buyListing)
	mux.HandleFunc("DELETE /api/resale/{id}", h.cancelListing)
}

type ResaleListing struct {
	ID          string    `json:"id"`
	TicketID    string    `json:"ticket_id"`
	SellerID    string    `json:"seller_id"`
	AskingPrice float64   `json:"asking_price"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

func (h *ResaleHandler) listTicket(w http.ResponseWriter, r *http.Request) {
	ticketID := r.PathValue("id")
	userID := r.Header.Get("X-User-ID")

	var body struct {
		AskingPrice float64 `json:"asking_price"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	var listing ResaleListing
	err := h.db.QueryRow(
		`INSERT INTO resale_listings (ticket_id, seller_id, asking_price)
		 VALUES ($1,$2,$3) RETURNING id, ticket_id, seller_id, asking_price, status, created_at, expires_at`,
		ticketID, userID, body.AskingPrice,
	).Scan(&listing.ID, &listing.TicketID, &listing.SellerID, &listing.AskingPrice,
		&listing.Status, &listing.CreatedAt, &listing.ExpiresAt)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, listing)
}

func (h *ResaleHandler) browseListings(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("id")

	rows, err := h.db.Query(
		`SELECT rl.id, rl.ticket_id, rl.seller_id, rl.asking_price, rl.status, rl.created_at, rl.expires_at
		 FROM resale_listings rl
		 JOIN tickets t ON t.id = rl.ticket_id
		 WHERE t.event_id = $1 AND rl.status = 'active' AND rl.expires_at > now()
		 ORDER BY rl.asking_price ASC`, eventID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	var listings []ResaleListing
	for rows.Next() {
		var l ResaleListing
		if err := rows.Scan(&l.ID, &l.TicketID, &l.SellerID, &l.AskingPrice,
			&l.Status, &l.CreatedAt, &l.ExpiresAt); err != nil {
			continue
		}
		listings = append(listings, l)
	}
	if listings == nil {
		listings = []ResaleListing{}
	}
	writeJSON(w, http.StatusOK, listings)
}

func (h *ResaleHandler) buyListing(w http.ResponseWriter, r *http.Request) {
	listingID := r.PathValue("id")
	buyerID := r.Header.Get("X-User-ID")

	var body struct {
		ZKProof string `json:"zk_proof"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	// Verify ZK proof
	if ok, err := verifyZKProof(body.ZKProof); err != nil || !ok {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "ZK proof verification failed"})
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
		return
	}
	defer tx.Rollback()

	var listing ResaleListing
	err = tx.QueryRow(
		`SELECT id, ticket_id, seller_id, asking_price, status
		 FROM resale_listings WHERE id = $1 AND status = 'active' AND expires_at > now()
		 FOR UPDATE`, listingID,
	).Scan(&listing.ID, &listing.TicketID, &listing.SellerID, &listing.AskingPrice, &listing.Status)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "listing not available"})
		return
	}

	if listing.SellerID == buyerID {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cannot buy your own listing"})
		return
	}

	_, err = tx.Exec(`UPDATE resale_listings SET status = 'sold' WHERE id = $1`, listingID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update failed"})
		return
	}

	_, err = tx.Exec(
		`INSERT INTO resale_transactions (listing_id, buyer_id, price, transfer_proof)
		 VALUES ($1,$2,$3,$4)`,
		listingID, buyerID, listing.AskingPrice, body.ZKProof)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "transaction failed"})
		return
	}

	_, err = tx.Exec(`UPDATE tickets SET owner_id = $1 WHERE id = $2`, buyerID, listing.TicketID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "ticket transfer failed"})
		return
	}

	tx.Commit()
	writeJSON(w, http.StatusOK, map[string]string{"status": "purchased", "ticket_id": listing.TicketID})
}

func (h *ResaleHandler) cancelListing(w http.ResponseWriter, r *http.Request) {
	listingID := r.PathValue("id")
	userID := r.Header.Get("X-User-ID")

	_, err := h.db.Exec(
		`UPDATE resale_listings SET status = 'cancelled'
		 WHERE id = $1 AND seller_id = $2 AND status = 'active'`,
		listingID, userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}
