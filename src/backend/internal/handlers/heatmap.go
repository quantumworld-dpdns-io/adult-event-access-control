// TODO: Register this handler in cmd/server/main.go
package handlers

import (
	"database/sql"
	"net/http"
)

type HeatmapHandler struct {
	db *sql.DB
}

type SectionCapacity struct {
	Section    string  `json:"section"`
	Capacity   int     `json:"capacity"`
	Occupied   int     `json:"occupied"`
	CheckIns   int     `json:"checkins"`
	Percentage float64 `json:"percentage"`
}

type OccupancySnapshot struct {
	Timestamp string  `json:"timestamp"`
	Occupied  int     `json:"occupied"`
	Capacity  int     `json:"capacity"`
	Percentage float64 `json:"percentage"`
}

func NewHeatmapHandler(db *sql.DB) *HeatmapHandler {
	return &HeatmapHandler{db: db}
}

func (h *HeatmapHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/events/{id}/heatmap", h.getHeatmap)
	mux.HandleFunc("GET /api/events/{id}/heatmap/history", h.getHistory)
}

func (h *HeatmapHandler) getHeatmap(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("id")

	rows, err := h.db.Query(
		`SELECT vs.section_name, vs.capacity,
			COALESCE(occ.checked_in, 0) AS occupied,
			COALESCE(occ.total_tickets, 0) AS total_tickets
		FROM venue_sections vs
		LEFT JOIN (
			SELECT t.event_id,
				COUNT(*) FILTER (WHERE t.status = 'checked_in') AS checked_in,
				COUNT(*) AS total_tickets
			FROM tickets t
			WHERE t.event_id = $1
			GROUP BY t.event_id
		) occ ON occ.event_id = vs.event_id
		WHERE vs.event_id = $1
		ORDER BY vs.sort_order, vs.section_name`, eventID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	var sections []SectionCapacity
	for rows.Next() {
		var s SectionCapacity
		if err := rows.Scan(&s.Section, &s.Capacity, &s.Occupied, &s.CheckIns); err != nil {
			continue
		}
		s.CheckIns = s.Occupied
		if s.Capacity > 0 {
			s.Percentage = float64(s.Occupied) / float64(s.Capacity) * 100
		}
		sections = append(sections, s)
	}
	if sections == nil {
		sections = []SectionCapacity{}
	}
	writeJSON(w, http.StatusOK, sections)
}

func (h *HeatmapHandler) getHistory(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("id")

	rows, err := h.db.Query(
		`SELECT DATE_TRUNC('minute', created_at) AS bucket,
			COUNT(*) AS checked_in
		FROM check_in_log
		WHERE event_id = $1
		GROUP BY bucket
		ORDER BY bucket DESC
		LIMIT 120`, eventID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	totalRows, err := h.db.Query(
		`SELECT COALESCE(SUM(capacity), 0) FROM venue_sections WHERE event_id = $1`, eventID)
	var totalCapacity int
	if err == nil {
		totalRows.Next()
		totalRows.Scan(&totalCapacity)
		totalRows.Close()
	}
	if totalCapacity == 0 {
		totalCapacity = 1
	}

	var snapshots []OccupancySnapshot
	var runningTotal int
	for rows.Next() {
		var bucket string
		var count int
		if err := rows.Scan(&bucket, &count); err != nil {
			continue
		}
		runningTotal += count
		snapshots = append(snapshots, OccupancySnapshot{
			Timestamp:  bucket,
			Occupied:   runningTotal,
			Capacity:   totalCapacity,
			Percentage: float64(runningTotal) / float64(totalCapacity) * 100,
		})
	}

	if snapshots == nil {
		snapshots = []OccupancySnapshot{}
	}
	writeJSON(w, http.StatusOK, snapshots)
}
