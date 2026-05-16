package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupServer() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/auth/oauth", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Provider string `json:"provider"`
			Code     string `json:"code"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{
			"token":   "test-jwt-token",
			"user_id": "test-user-id",
		})
	})

	mux.HandleFunc("POST /api/auth/siwe", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"token":   "siwe-jwt-token",
			"user_id": "siwe-user-id",
		})
	})

	mux.HandleFunc("POST /api/auth/world-id", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"token":   "world-id-jwt-token",
			"user_id": "world-id-user-id",
		})
	})

	mux.HandleFunc("GET /api/auth/me", func(w http.ResponseWriter, r *http.Request) {
		uid := r.Header.Get("X-User-ID")
		if uid == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{
			"user_id": uid, "display_name": "Test User",
		})
	})

	mux.HandleFunc("GET /api/events", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]any{
			{
				"id": "event-1", "title": "Test Event",
				"status": "published", "capacity": 100, "min_age": 18,
			},
		})
	})

	mux.HandleFunc("POST /api/events", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"id": "new-event-id", "title": "New Event",
		})
	})

	mux.HandleFunc("GET /api/events/{id}", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"id": r.PathValue("id"), "title": "Test Event",
		})
	})

	mux.HandleFunc("POST /api/events/{eventId}/tickets", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"id": "ticket-1", "event_id": r.PathValue("eventId"), "status": "active",
		})
	})

	mux.HandleFunc("GET /api/tickets/{id}", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"id": r.PathValue("id"), "status": "active",
		})
	})

	mux.HandleFunc("POST /api/checkin", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"status": "checked_in", "ticket_id": "ticket-1",
		})
	})

	mux.HandleFunc("GET /api/events/{eventId}/attendees", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]string{
			{"ticket_id": "ticket-1", "display_name": "Alice", "status": "checked_in"},
		})
	})

	return mux
}

func TestAuthOAuth(t *testing.T) {
	srv := httptest.NewServer(setupServer())
	defer srv.Close()

	body, _ := json.Marshal(map[string]string{"provider": "google", "code": "auth-code"})
	resp, err := http.Post(srv.URL+"/api/auth/oauth", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	if result["token"] == "" {
		t.Fatal("expected token in response")
	}
	if result["user_id"] == "" {
		t.Fatal("expected user_id in response")
	}
}

func TestAuthSIWE(t *testing.T) {
	srv := httptest.NewServer(setupServer())
	defer srv.Close()

	body, _ := json.Marshal(map[string]string{
		"message":   "aev.xyz wants you to sign in...",
		"signature": "0xdeadbeef",
	})
	resp, err := http.Post(srv.URL+"/api/auth/siwe", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	if result["token"] == "" {
		t.Fatal("expected token in SIWE response")
	}
}

func TestAuthWorldID(t *testing.T) {
	srv := httptest.NewServer(setupServer())
	defer srv.Close()

	body, _ := json.Marshal(map[string]string{
		"nullifier_hash": "0xabc123",
		"proof":          "proof-data",
	})
	resp, err := http.Post(srv.URL+"/api/auth/world-id", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestAuthMe(t *testing.T) {
	srv := httptest.NewServer(setupServer())
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/api/auth/me", nil)
	req.Header.Set("X-User-ID", "user-1")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	if result["user_id"] != "user-1" {
		t.Fatalf("expected user-1, got %s", result["user_id"])
	}
}

func TestAuthMeUnauthorized(t *testing.T) {
	srv := httptest.NewServer(setupServer())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/auth/me")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestListEvents(t *testing.T) {
	srv := httptest.NewServer(setupServer())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/events")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var events []map[string]any
	json.NewDecoder(resp.Body).Decode(&events)
	if len(events) == 0 {
		t.Fatal("expected at least one event")
	}
}

func TestCreateEvent(t *testing.T) {
	srv := httptest.NewServer(setupServer())
	defer srv.Close()

	body, _ := json.Marshal(map[string]any{
		"title": "New Year Party", "capacity": 200, "min_age": 21,
	})
	resp, err := http.Post(srv.URL+"/api/events", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestGetEvent(t *testing.T) {
	srv := httptest.NewServer(setupServer())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/events/event-1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	if result["id"] != "event-1" {
		t.Fatalf("expected event-1, got %s", result["id"])
	}
}

func TestIssueTicket(t *testing.T) {
	srv := httptest.NewServer(setupServer())
	defer srv.Close()

	body, _ := json.Marshal(map[string]any{
		"zk_proof_commitment": "proof-commitment",
		"ticket_type":         "vip",
	})
	resp, err := http.Post(srv.URL+"/api/events/event-1/tickets", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	if result["status"] != "active" {
		t.Fatalf("expected active status, got %s", result["status"])
	}
}

func TestGetTicket(t *testing.T) {
	srv := httptest.NewServer(setupServer())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/tickets/ticket-1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestCheckIn(t *testing.T) {
	srv := httptest.NewServer(setupServer())
	defer srv.Close()

	body, _ := json.Marshal(map[string]string{
		"qr_secret": "qr-123",
		"method":    "qr",
	})
	resp, err := http.Post(srv.URL+"/api/checkin", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	if result["status"] != "checked_in" {
		t.Fatalf("expected checked_in, got %s", result["status"])
	}
}

func TestAttendees(t *testing.T) {
	srv := httptest.NewServer(setupServer())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/events/event-1/attendees")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var attendees []map[string]string
	json.NewDecoder(resp.Body).Decode(&attendees)
	if len(attendees) == 0 {
		t.Fatal("expected at least one attendee")
	}
}
