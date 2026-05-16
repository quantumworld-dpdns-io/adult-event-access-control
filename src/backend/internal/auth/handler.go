package auth

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/auth/oauth", h.oauth)
	mux.HandleFunc("POST /api/auth/siwe", h.siwe)
	mux.HandleFunc("POST /api/auth/world-id", h.worldID)
	mux.HandleFunc("GET /api/auth/me", h.me)
}

type authResponse struct {
	Token   string `json:"token"`
	UserID  string `json:"user_id"`
	Display string `json:"display_name,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) oauth(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Provider string `json:"provider"` // google, github
		Code     string `json:"code"`
		State    string `json:"state,omitempty"`
		Redirect string `json:"redirect_uri,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	userInfo, err := verifyOAuth(body.Provider, body.Code, body.State, body.Redirect)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	uid, err := h.upsertUser(userInfo)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "user creation failed"})
		return
	}

	token, _ := issueToken(uid)
	writeJSON(w, http.StatusOK, authResponse{Token: token, UserID: uid})
}

func (h *Handler) siwe(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Message   string `json:"message"`
		Signature string `json:"signature"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	address, err := verifySIWE(body.Message, body.Signature)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	uid, err := h.upsertUser(&userInfo{Provider: "siwe", ProviderID: address})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "user creation failed"})
		return
	}

	token, _ := issueToken(uid)
	writeJSON(w, http.StatusOK, authResponse{Token: token, UserID: uid})
}

func (h *Handler) worldID(w http.ResponseWriter, r *http.Request) {
	var body struct {
		NullifierHash string `json:"nullifier_hash"`
		Proof         string `json:"proof"`
		MerkleRoot    string `json:"merkle_root,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	appID := r.Header.Get("X-WorldID-App")
	if appID == "" {
		appID = "app_staging"
	}

	valid, err := verifyWorldID(body.Proof, appID, body.NullifierHash)
	if err != nil || !valid {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "world id verification failed"})
		return
	}

	uid, err := h.upsertUser(&userInfo{Provider: "world_id", ProviderID: body.NullifierHash})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "user creation failed"})
		return
	}

	token, _ := issueToken(uid)
	writeJSON(w, http.StatusOK, authResponse{Token: token, UserID: uid})
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	uid := r.Header.Get("X-User-ID")
	if uid == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var display string
	err := h.db.QueryRow("SELECT display_name FROM users WHERE id = $1", uid).Scan(&display)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"user_id": uid, "display_name": display})
}

type userInfo struct {
	Provider   string
	ProviderID string
	Email      string
	Display    string
	Avatar     string
}

func (h *Handler) upsertUser(u *userInfo) (string, error) {
	tx, err := h.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	var uid string
	err = tx.QueryRow(`SELECT user_id FROM auth_providers WHERE provider = $1 AND provider_id = $2`,
		u.Provider, u.ProviderID).Scan(&uid)
	if err == nil {
		tx.Commit()
		return uid, nil
	}

	err = tx.QueryRow(`INSERT INTO users (email, display_name, avatar_url)
		VALUES ($1, $2, $3) RETURNING id`,
		nullIf(u.Email), nullIf(u.Display), nullIf(u.Avatar),
	).Scan(&uid)
	if err != nil {
		return "", err
	}

	_, err = tx.Exec(`INSERT INTO auth_providers (user_id, provider, provider_id)
		VALUES ($1, $2, $3)`, uid, u.Provider, u.ProviderID)
	if err != nil {
		return "", err
	}

	return uid, tx.Commit()
}

func nullIf(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
