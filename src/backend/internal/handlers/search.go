// TODO: Register this handler in cmd/server/main.go
package handlers

import (
	"database/sql"
	"net/http"

	"github.com/quantumworld-dpdns-io/adult-event-access-control/backend/internal/search"
)

// SearchHandler handles full-text + vector search across events.
type SearchHandler struct {
	db           *sql.DB
	searchClient *search.QdrantClient
}

// NewSearchHandler creates a SearchHandler.
func NewSearchHandler(db *sql.DB, sc *search.QdrantClient) *SearchHandler {
	return &SearchHandler{db: db, searchClient: sc}
}

// RegisterRoutes registers search endpoints on the given mux.
func (h *SearchHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/search/events", h.search)
}

// search performs full-text search on PostgreSQL and merges Qdrant results.
func (h *SearchHandler) search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "query parameter 'q' is required"})
		return
	}

	limit := 20

	// PostgreSQL full-text search via ts_vector / ts_rank.
	rows, err := h.db.Query(
		`SELECT id, title, ts_rank(to_tsvector('english', title || ' ' || COALESCE(description,'')), plainto_tsquery('english', $1)) AS rank
		 FROM events
		 WHERE to_tsvector('english', title || ' ' || COALESCE(description,'')) @@ plainto_tsquery('english', $1)
		 ORDER BY rank DESC
		 LIMIT $2`,
		q, limit,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	type result struct {
		ID    string  `json:"id"`
		Title string  `json:"title"`
		Score float64 `json:"rank"`
	}

	var pgResults []result
	for rows.Next() {
		var r result
		if err := rows.Scan(&r.ID, &r.Title, &r.Score); err != nil {
			continue
		}
		pgResults = append(pgResults, r)
	}
	if pgResults == nil {
		pgResults = []result{}
	}

	// Merge with Qdrant vector search results.
	vecResults, _ := h.searchClient.SearchEvents(q, limit)

	seen := make(map[string]bool, len(pgResults))
	merged := make([]result, 0, len(pgResults)+len(vecResults))
	for _, r := range pgResults {
		seen[r.ID] = true
		merged = append(merged, r)
	}
	for _, vr := range vecResults {
		if !seen[vr.ID] {
			merged = append(merged, result{ID: vr.ID, Title: vr.Title, Score: vr.Score})
			seen[vr.ID] = true
		}
	}

	writeJSON(w, http.StatusOK, merged)
}
