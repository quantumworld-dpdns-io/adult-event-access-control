// TODO: Register this handler in cmd/server/main.go
package handlers

import (
	"database/sql"
	"math/rand"
	"net/http"
)

type RecommendHandler struct {
	db *sql.DB
}

func NewRecommendHandler(db *sql.DB) *RecommendHandler {
	return &RecommendHandler{db: db}
}

func (h *RecommendHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/recommend/events", h.recommendEvents)
	mux.HandleFunc("GET /api/events/{id}/similar", h.similarEvents)
	mux.HandleFunc("GET /api/tickets/{id}/noshow-risk", h.noshowRisk)
}

// TODO: Replace placeholder logic with real ML inference from event_embeddings and user_preferences tables
func (h *RecommendHandler) recommendEvents(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")

	var category string
	err := h.db.QueryRow(
		`SELECT COALESCE(e.category, '')
		 FROM tickets t
		 JOIN events e ON e.id = t.event_id
		 WHERE t.owner_id = $1 AND t.status IN ('active','checked_in')
		 GROUP BY e.category
		 ORDER BY COUNT(*) DESC
		 LIMIT 1`, userID,
	).Scan(&category)
	if err != nil {
		category = ""
	}

	query := `SELECT id, title, COALESCE(category,'') AS category, start_time
		FROM events WHERE status = 'published'`
	args := []any{}
	if category != "" {
		query += ` AND category = $1`
		args = append(args, category)
	}
	query += ` ORDER BY start_time ASC LIMIT 5`

	rows, err := h.db.Query(query, args...)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	var results []map[string]any
	for rows.Next() {
		var id, title, cat string
		var startTime interface{}
		if err := rows.Scan(&id, &title, &cat, &startTime); err != nil {
			continue
		}
		results = append(results, map[string]any{
			"event_id":   id,
			"title":      title,
			"category":   cat,
			"start_time": startTime,
			"score":      0.5 + rand.Float64()*0.5,
		})
	}
	if results == nil {
		results = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, results)
}

// TODO: Replace with real embedding similarity query (e.g. pgvector cosine similarity)
func (h *RecommendHandler) similarEvents(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("id")

	var category string
	var startTime interface{}
	err := h.db.QueryRow(
		`SELECT COALESCE(category,''), start_time FROM events WHERE id = $1`,
		eventID,
	).Scan(&category, &startTime)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "event not found"})
		return
	}

	rows, err := h.db.Query(
		`SELECT id, title, COALESCE(category,''), start_time
		 FROM events WHERE id != $1 AND status = 'published' AND category = $2
		 ORDER BY start_time ASC LIMIT 5`,
		eventID, category,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	var results []map[string]any
	for rows.Next() {
		var id, title, cat string
		var st interface{}
		if err := rows.Scan(&id, &title, &cat, &st); err != nil {
			continue
		}
		results = append(results, map[string]any{
			"event_id":   id,
			"title":      title,
			"category":   cat,
			"start_time": st,
			"similarity": 0.6 + rand.Float64()*0.4,
		})
	}
	if results == nil {
		results = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, results)
}

// TODO: Replace with real ML model inference from noshow_predictions table
func (h *RecommendHandler) noshowRisk(w http.ResponseWriter, r *http.Request) {
	ticketID := r.PathValue("id")

	var eventID string
	var issuedAt, startTime interface{}
	err := h.db.QueryRow(
		`SELECT t.event_id, t.issued_at, e.start_time
		 FROM tickets t JOIN events e ON e.id = t.event_id
		 WHERE t.id = $1`, ticketID,
	).Scan(&eventID, &issuedAt, &startTime)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "ticket not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ticket_id":   ticketID,
		"event_id":    eventID,
		"probability": 0.1 + rand.Float64()*0.3,
		"risk_level":  "low",
		"note":        "placeholder — integrate noshow_predictions table for real inference",
	})
}
