// TODO: Register this handler in cmd/server/main.go
package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/lib/pq"
)

type Alert struct {
	ID        string    `json:"id,omitempty"`
	Type      string    `json:"type"`
	EventID   string    `json:"event_id"`
	Message   string    `json:"message"`
	Severity  string    `json:"severity"`
	Timestamp time.Time `json:"timestamp"`
}

type AlertHandler struct {
	db          *sql.DB
	dsn         string
	subscribers map[string][]chan Alert
	mu          sync.RWMutex
	listener    *pq.Listener
	ctx         context.Context
	cancel      context.CancelFunc
}

func NewAlertHandler(db *sql.DB, dsn string) *AlertHandler {
	ctx, cancel := context.WithCancel(context.Background())
	return &AlertHandler{
		db:          db,
		dsn:         dsn,
		subscribers: make(map[string][]chan Alert),
		ctx:         ctx,
		cancel:      cancel,
	}
}

func (h *AlertHandler) Start() {
	h.listener = pq.NewListener(h.dsn, 10*time.Second, time.Minute, func(ev pq.ListenerEventType, err error) {
		if err != nil {
			log.Printf("alert listener error (event=%d): %v", ev, err)
		}
	})

	if err := h.listener.Listen("alert_channel"); err != nil {
		log.Printf("failed to LISTEN alert_channel: %v", err)
		return
	}

	go h.listenLoop()
}

func (h *AlertHandler) Stop() {
	h.cancel()
	if h.listener != nil {
		h.listener.Close()
	}
}

func (h *AlertHandler) listenLoop() {
	for {
		select {
		case <-h.ctx.Done():
			return
		case n := <-h.listener.Notify:
			if n == nil {
				continue
			}
			var alert Alert
			if err := json.Unmarshal([]byte(n.Extra), &alert); err != nil {
				continue
			}
			h.mu.RLock()
			chans := h.subscribers[alert.EventID]
			h.mu.RUnlock()
			for _, ch := range chans {
				select {
				case ch <- alert:
				default:
				}
			}
		case <-time.After(90 * time.Second):
			go h.listener.Ping()
		}
	}
}

func (h *AlertHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/events/{id}/alerts", h.sseAlerts)
	mux.HandleFunc("POST /api/events/{id}/alerts", h.createAlert)
	mux.HandleFunc("GET /api/events/{id}/alerts/history", h.alertHistory)
}

func (h *AlertHandler) sseAlerts(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("id")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming not supported"})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan Alert, 64)

	h.mu.Lock()
	h.subscribers[eventID] = append(h.subscribers[eventID], ch)
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		subs := h.subscribers[eventID]
		for i, c := range subs {
			if c == ch {
				h.subscribers[eventID] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
		if len(h.subscribers[eventID]) == 0 {
			delete(h.subscribers, eventID)
		}
		h.mu.Unlock()
		close(ch)
	}()

	fmt.Fprintf(w, "data: {\"type\":\"connected\",\"event_id\":%q}\n\n", eventID)
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case alert, ok := <-ch:
			if !ok {
				return
			}
			data, _ := json.Marshal(alert)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func (h *AlertHandler) createAlert(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("id")

	var body struct {
		Type     string `json:"type"`
		Message  string `json:"message"`
		Severity string `json:"severity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if body.Type == "" {
		body.Type = "general"
	}
	if body.Severity == "" {
		body.Severity = "info"
	}
	if body.Message == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "message is required"})
		return
	}

	var alert Alert
	err := h.db.QueryRow(
		`INSERT INTO alerts (event_id, type, message, severity)
		VALUES ($1, $2, $3, $4) RETURNING id, event_id, type, message, severity, created_at`,
		eventID, body.Type, body.Message, body.Severity,
	).Scan(&alert.ID, &alert.EventID, &alert.Type, &alert.Message, &alert.Severity, &alert.Timestamp)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	alert.Timestamp = alert.Timestamp.UTC()

	writeJSON(w, http.StatusCreated, alert)
}

func (h *AlertHandler) alertHistory(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("id")

	rows, err := h.db.Query(
		`SELECT id, event_id, type, message, severity, created_at
		FROM alerts WHERE event_id = $1 ORDER BY created_at DESC LIMIT 100`, eventID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	var alerts []Alert
	for rows.Next() {
		var a Alert
		if err := rows.Scan(&a.ID, &a.EventID, &a.Type, &a.Message, &a.Severity, &a.Timestamp); err != nil {
			continue
		}
		a.Timestamp = a.Timestamp.UTC()
		alerts = append(alerts, a)
	}
	if alerts == nil {
		alerts = []Alert{}
	}
	writeJSON(w, http.StatusOK, alerts)
}
