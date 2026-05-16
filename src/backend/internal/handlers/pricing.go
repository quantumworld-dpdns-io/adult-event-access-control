// TODO: Register this handler in cmd/server/main.go
package handlers

import (
	"database/sql"
	"encoding/json"
	"math"
	"net/http"
	"time"
)

type PricingHandler struct {
	db *sql.DB
}

func NewPricingHandler(db *sql.DB) *PricingHandler {
	return &PricingHandler{db: db}
}

func (h *PricingHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/events/{id}/pricing", h.getPricing)
	mux.HandleFunc("POST /api/events/{id}/pricing/tiers", h.createTier)
}

type PricingTier struct {
	ID        string  `json:"id,omitempty"`
	EventID   string  `json:"event_id"`
	TierName  string  `json:"tier_name"`
	Price     float64 `json:"price"`
	Capacity  int     `json:"capacity"`
	SortOrder int     `json:"sort_order"`
}

func (h *PricingHandler) getPricing(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("id")

	var basePrice float64
	var capacity, ticketsSold int
	var startTime time.Time
	err := h.db.QueryRow(
		`SELECT COALESCE(MIN(pt.price), 0.0), e.capacity,
		        COALESCE((SELECT COUNT(*) FROM tickets WHERE event_id = $1 AND status != 'cancelled'), 0),
		        e.start_time
		 FROM events e
		 LEFT JOIN pricing_tiers pt ON pt.event_id = e.id
		 WHERE e.id = $1
		 GROUP BY e.capacity, e.start_time`, eventID,
	).Scan(&basePrice, &capacity, &ticketsSold, &startTime)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "event not found"})
		return
	}

	ratio := float64(ticketsSold) / float64(capacity)
	hoursUntilEvent := time.Until(startTime).Hours()

	surgeMultiplier := 1.0
	var demandLevel string

	if ratio > 0.9 {
		surgeMultiplier = 2.0
		demandLevel = "very_high"
	} else if ratio > 0.75 {
		surgeMultiplier = 1.5
		demandLevel = "high"
	} else if ratio > 0.5 {
		surgeMultiplier = 1.2
		demandLevel = "medium"
	} else {
		demandLevel = "low"
	}

	if hoursUntilEvent < 24 && ratio > 0.5 {
		surgeMultiplier += 0.3
	} else if hoursUntilEvent < 72 && ratio > 0.7 {
		surgeMultiplier += 0.2
	}

	surgeMultiplier = math.Round(surgeMultiplier*100) / 100

	writeJSON(w, http.StatusOK, map[string]any{
		"base_price":       basePrice,
		"surge_multiplier": surgeMultiplier,
		"current_price":    math.Round(basePrice*surgeMultiplier*100) / 100,
		"tickets_remaining": capacity - ticketsSold,
		"demand_level":     demandLevel,
	})
}

func (h *PricingHandler) createTier(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("id")

	var tier PricingTier
	if err := json.NewDecoder(r.Body).Decode(&tier); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	tier.EventID = eventID
	err := h.db.QueryRow(
		`INSERT INTO pricing_tiers (event_id, tier_name, price, capacity, sort_order)
		 VALUES ($1,$2,$3,$4,$5) RETURNING id`,
		tier.EventID, tier.TierName, tier.Price, tier.Capacity, tier.SortOrder,
	).Scan(&tier.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, tier)
}
