// TODO: Register this handler in cmd/server/main.go
package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/quantumworld-dpdns-io/adult-event-access-control/backend/internal/pqc"
)

type PQCHandler struct {
	db          *sql.DB
	pqcVerifier *pqc.MLDSA
}

func NewPQCHandler(db *sql.DB) *PQCHandler {
	return &PQCHandler{db: db, pqcVerifier: pqc.NewMLDSA()}
}

func (h *PQCHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/pqc/identity", h.createIdentity)
	mux.HandleFunc("POST /api/pqc/verify", h.verifyProof)
	mux.HandleFunc("GET /api/pqc/keypair", h.keypair)
}

// createIdentity creates a PQC identity credential for the user
func (h *PQCHandler) createIdentity(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	cred, err := pqc.GeneratePQCIdentity(body.UserID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create identity"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":    true,
		"credential": cred.Marshal(),
	})
}

// verifyProof verifies a PQC proof
func (h *PQCHandler) verifyProof(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TicketID string `json:"ticket_id"`
		UserID   string `json:"user_id"`
		Proof    string `json:"proof"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	verified, err := pqc.VerifyPQCProof(body.TicketID, body.UserID, body.Proof)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"success":  false,
			"verified": false,
			"error":    err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":  true,
		"verified": verified,
	})
}

// keypair generates and returns a new ML-DSA keypair
func (h *PQCHandler) keypair(w http.ResponseWriter, r *http.Request) {
	mldsa := pqc.NewMLDSA()
	pk, sk := mldsa.GenerateKeypair()

	writeJSON(w, http.StatusOK, map[string]string{
		"success":    "true",
		"public_key": hexEncode(pk),
		"secret_key": hexEncode(sk),
		"algorithm":  "ML-DSA-65",
	})
}

func hexEncode(b []byte) string {
	const hexChars = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, v := range b {
		out[i*2] = hexChars[v>>4]
		out[i*2+1] = hexChars[v&0x0f]
	}
	return string(out)
}
