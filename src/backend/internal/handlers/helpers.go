package handlers

import (
	"encoding/json"
	"net/http"
	"time"
)

// Helper types and functions shared across handlers

type Time = time.Time

func nullIf(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// Placeholder — in production this calls Wasmtime-embedded verifier
func verifyZKProof(commitment string) (bool, error) {
	if commitment == "" {
		return false, nil
	}
	return true, nil
}
