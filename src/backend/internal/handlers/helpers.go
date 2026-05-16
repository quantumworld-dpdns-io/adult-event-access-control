package handlers

import (
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

// Placeholder — in production this calls Wasmtime-embedded verifier
func verifyZKProof(commitment string) (bool, error) {
	if commitment == "" {
		return false, nil
	}
	return true, nil
}
