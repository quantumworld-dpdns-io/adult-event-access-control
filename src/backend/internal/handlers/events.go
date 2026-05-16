package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

type EventHandler struct {
	db *sql.DB
}

func NewEventHandler(db *sql.DB) *EventHandler {
	return &EventHandler{db: db}
}

func (h *EventHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/events", h.list)
	mux.HandleFunc("POST /api/events", h.create)
	mux.HandleFunc("GET /api/events/{id}", h.get)
	mux.HandleFunc("PUT /api/events/{id}", h.update)
	mux.HandleFunc("DELETE /api/events/{id}", h.delete)
}

type Event struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	VenueName   string    `json:"venue_name,omitempty"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Capacity    int       `json:"capacity"`
	MinAge      int       `json:"min_age"`
	Status      string    `json:"status"`
	Category    string    `json:"category,omitempty"`
}

func (h *EventHandler) list(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "published"
	}

	query := `SELECT id, title, COALESCE(description,''), COALESCE(venue_name,''),
		start_time, end_time, capacity, min_age, status, COALESCE(category,'')
		FROM events WHERE status = $1`
	args := []any{status}

	if category != "" {
		query += ` AND category = $2`
		args = append(args, category)
	}
	query += ` ORDER BY start_time ASC`

	rows, err := h.db.Query(query, args...)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.Title, &e.Description, &e.VenueName,
			&e.StartTime, &e.EndTime, &e.Capacity, &e.MinAge, &e.Status, &e.Category); err != nil {
			continue
		}
		events = append(events, e)
	}
	if events == nil {
		events = []Event{}
	}
	writeJSON(w, http.StatusOK, events)
}

func (h *EventHandler) create(w http.ResponseWriter, r *http.Request) {
	var e Event
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	err := h.db.QueryRow(
		`INSERT INTO events (organizer_id, title, description, venue_name, start_time, end_time, capacity, min_age, category)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id`,
		r.Header.Get("X-User-ID"), e.Title, e.Description, e.VenueName,
		e.StartTime, e.EndTime, e.Capacity, e.MinAge, e.Category,
	).Scan(&e.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

func (h *EventHandler) get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var e Event
	err := h.db.QueryRow(
		`SELECT id, title, COALESCE(description,''), COALESCE(venue_name,''),
		start_time, end_time, capacity, min_age, status, COALESCE(category,'')
		FROM events WHERE id = $1`, id,
	).Scan(&e.ID, &e.Title, &e.Description, &e.VenueName,
		&e.StartTime, &e.EndTime, &e.Capacity, &e.MinAge, &e.Status, &e.Category)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "event not found"})
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (h *EventHandler) update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var e Event
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	_, err := h.db.Exec(
		`UPDATE events SET title=$1, description=$2, venue_name=$3, start_time=$4,
		end_time=$5, capacity=$6, min_age=$7, category=$8, updated_at=now()
		WHERE id=$9 AND organizer_id=$10`,
		e.Title, e.Description, e.VenueName, e.StartTime, e.EndTime,
		e.Capacity, e.MinAge, e.Category, id, r.Header.Get("X-User-ID"),
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	e.ID = id
	writeJSON(w, http.StatusOK, e)
}

func (h *EventHandler) delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	_, err := h.db.Exec(`DELETE FROM events WHERE id = $1 AND organizer_id = $2`,
		id, r.Header.Get("X-User-ID"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
