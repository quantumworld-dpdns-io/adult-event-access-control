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

type WSClient struct {
	conn    http.ResponseWriter
	flusher http.Flusher
	eventID string
	send    chan []byte
	done    chan struct{}
}

type WSHandler struct {
	db      *sql.DB
	dsn     string
	clients map[string]map[*WSClient]bool
	mu      sync.RWMutex
	listener *pq.Listener
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewWSHandler(db *sql.DB, dsn string) *WSHandler {
	ctx, cancel := context.WithCancel(context.Background())
	return &WSHandler{
		db:      db,
		dsn:     dsn,
		clients: make(map[string]map[*WSClient]bool),
		ctx:     ctx,
		cancel:  cancel,
	}
}

func (h *WSHandler) Start() {
	h.listener = pq.NewListener(h.dsn, 10*time.Second, time.Minute, func(ev pq.ListenerEventType, err error) {
		if err != nil {
			log.Printf("ws listener error (event=%d): %v", ev, err)
		}
	})

	if err := h.listener.Listen("venue_update"); err != nil {
		log.Printf("failed to LISTEN venue_update: %v", err)
		return
	}

	go h.listenLoop()
}

func (h *WSHandler) Stop() {
	h.cancel()
	if h.listener != nil {
		h.listener.Close()
	}
}

func (h *WSHandler) listenLoop() {
	for {
		select {
		case <-h.ctx.Done():
			return
		case n := <-h.listener.Notify:
			if n == nil {
				continue
			}
			var payload struct {
				EventID   string    `json:"event_id"`
				Type      string    `json:"type"`
				Timestamp time.Time `json:"timestamp"`
			}
			if err := json.Unmarshal([]byte(n.Extra), &payload); err != nil {
				continue
			}
			msg, _ := json.Marshal(payload)
			h.mu.RLock()
			clients := h.clients[payload.EventID]
			h.mu.RUnlock()
			for c := range clients {
				select {
				case c.send <- msg:
				default:
				}
			}
		case <-time.After(90 * time.Second):
			go h.listener.Ping()
		}
	}
}

func (h *WSHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/events/{id}/live", h.sseLive)
	mux.HandleFunc("POST /api/events/{id}/broadcast", h.broadcast)
}

func (h *WSHandler) sseLive(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("id")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming not supported"})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	client := &WSClient{
		conn:    w,
		flusher: flusher,
		eventID: eventID,
		send:    make(chan []byte, 64),
		done:    make(chan struct{}),
	}

	h.mu.Lock()
	if h.clients[eventID] == nil {
		h.clients[eventID] = make(map[*WSClient]bool)
	}
	h.clients[eventID][client] = true
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.clients[eventID], client)
		if len(h.clients[eventID]) == 0 {
			delete(h.clients, eventID)
		}
		h.mu.Unlock()
		close(client.done)
	}()

	fmt.Fprintf(w, "data: {\"type\":\"connected\",\"event_id\":%q}\n\n", eventID)
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case msg, ok := <-client.send:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		}
	}
}

func (h *WSHandler) broadcast(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("id")

	var body struct {
		Type    string `json:"type"`
		Message string `json:"message,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if body.Type == "" {
		body.Type = "broadcast"
	}

	payload, _ := json.Marshal(map[string]any{
		"event_id": eventID,
		"type":     body.Type,
		"message":  body.Message,
		"timestamp": time.Now(),
	})

	_, err := h.db.Exec(fmt.Sprintf("NOTIFY venue_update, %s", pq.QuoteLiteral(string(payload))))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "broadcast sent"})
}
