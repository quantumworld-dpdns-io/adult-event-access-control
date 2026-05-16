package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// workflowServer simulates the full aeV backend for end-to-end workflow testing.
// In production these would be real Docker containers; here we use in-memory
// handlers to validate the full registration → event creation → ticket → check-in flow.
func workflowServer() http.Handler {
	mux := http.NewServeMux()

	// In-memory store
	type user struct {
		ID          string
		DisplayName string
		Token       string
	}
	type event struct {
		ID       string
		Title    string
		Capacity int
		MinAge   int
		Status   string
	}
	type ticket struct {
		ID      string
		EventID string
		OwnerID string
		Status  string
		QR      string
	}

	users := map[string]*user{}
	events := map[string]*event{}
	tickets := map[string]*ticket{}
	eventCount := 0
	ticketCount := 0

	mux.HandleFunc("POST /api/auth/world-id", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			NullifierHash string `json:"nullifier_hash"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		uid := "user-" + body.NullifierHash[:8]
		users[uid] = &user{ID: uid, DisplayName: "Alice", Token: "jwt-" + uid}
		json.NewEncoder(w).Encode(map[string]string{
			"token": users[uid].Token, "user_id": uid,
		})
	})

	mux.HandleFunc("POST /api/events", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Title    string `json:"title"`
			Capacity int    `json:"capacity"`
			MinAge   int    `json:"min_age"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		eventCount++
		eid := "event-" + itoa(eventCount)
		events[eid] = &event{
			ID: eid, Title: body.Title,
			Capacity: body.Capacity, MinAge: body.MinAge, Status: "published",
		}
		json.NewEncoder(w).Encode(events[eid])
	})

	mux.HandleFunc("GET /api/events/{id}", func(w http.ResponseWriter, r *http.Request) {
		eid := r.PathValue("id")
		e, ok := events[eid]
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(e)
	})

	mux.HandleFunc("POST /api/events/{eventId}/tickets", func(w http.ResponseWriter, r *http.Request) {
		eid := r.PathValue("eventId")
		uid := r.Header.Get("X-User-ID")
		ticketCount++
		tid := "ticket-" + itoa(ticketCount)
		tickets[tid] = &ticket{
			ID: tid, EventID: eid, OwnerID: uid, Status: "active", QR: "qr-" + tid,
		}
		json.NewEncoder(w).Encode(tickets[tid])
	})

	mux.HandleFunc("GET /api/tickets/{id}", func(w http.ResponseWriter, r *http.Request) {
		tid := r.PathValue("id")
		t, ok := tickets[tid]
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(t)
	})

	mux.HandleFunc("POST /api/checkin", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			QRSecret string `json:"qr_secret"`
		}
		json.NewDecoder(r.Body).Decode(&body)

		var found *ticket
		for _, t := range tickets {
			if t.QR == body.QRSecret && t.Status == "active" {
				found = t
				break
			}
		}
		if found == nil {
			http.Error(w, `{"error":"invalid ticket"}`, http.StatusNotFound)
			return
		}
		found.Status = "checked_in"
		json.NewEncoder(w).Encode(map[string]string{
			"status": "checked_in", "ticket_id": found.ID,
		})
	})

	return mux
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

func TestFullWorkflow(t *testing.T) {
	srv := httptest.NewServer(workflowServer())
	defer srv.Close()

	// 1. Register via World ID
	registerBody, _ := json.Marshal(map[string]string{
		"nullifier_hash": "0xabc123def456",
		"proof":          "world-proof",
	})
	resp, err := http.Post(srv.URL+"/api/auth/world-id", "application/json", bytes.NewReader(registerBody))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("register: expected 200, got %d", resp.StatusCode)
	}

	var authResp map[string]string
	json.NewDecoder(resp.Body).Decode(&authResp)
	resp.Body.Close()

	token := authResp["token"]
	userID := authResp["user_id"]
	if token == "" || userID == "" {
		t.Fatal("register: expected token and user_id")
	}
	t.Logf("Registered user: %s", userID)

	// 2. Create event
	eventBody, _ := json.Marshal(map[string]any{
		"title": "Summer Festival", "capacity": 500, "min_age": 18,
	})
	req, _ := http.NewRequest("POST", srv.URL+"/api/events", bytes.NewReader(eventBody))
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create event: expected 200, got %d", resp.StatusCode)
	}

	var createdEvent map[string]any
	json.NewDecoder(resp.Body).Decode(&createdEvent)
	resp.Body.Close()

	eventID := createdEvent["id"].(string)
	if eventID == "" {
		t.Fatal("create event: expected event id")
	}
	t.Logf("Created event: %s", eventID)

	// 3. Get ticket for event
	ticketBody, _ := json.Marshal(map[string]string{
		"zk_proof_commitment": "age-proof-commitment",
		"ticket_type":         "standard",
	})
	req, _ = http.NewRequest("POST", srv.URL+"/api/events/"+eventID+"/tickets", bytes.NewReader(ticketBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-User-ID", userID)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("issue ticket: expected 200, got %d", resp.StatusCode)
	}

	var issuedTicket map[string]any
	json.NewDecoder(resp.Body).Decode(&issuedTicket)
	resp.Body.Close()

	ticketID := issuedTicket["id"].(string)
	qrSecret := issuedTicket["qr"].(string)
	if ticketID == "" {
		t.Fatal("issue ticket: expected ticket id")
	}
	t.Logf("Issued ticket: %s (qr: %s)", ticketID, qrSecret)

	// 4. Check in with ticket QR
	checkinBody, _ := json.Marshal(map[string]string{
		"qr_secret": qrSecret,
		"method":    "qr",
	})
	resp, err = http.Post(srv.URL+"/api/checkin", "application/json", bytes.NewReader(checkinBody))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("checkin: expected 200, got %d", resp.StatusCode)
	}

	var checkinResp map[string]string
	json.NewDecoder(resp.Body).Decode(&checkinResp)
	resp.Body.Close()

	if checkinResp["status"] != "checked_in" {
		t.Fatalf("checkin: expected checked_in, got %s", checkinResp["status"])
	}
	t.Logf("Check-in successful for ticket: %s", checkinResp["ticket_id"])

	// 5. Verify ticket status is now checked_in
	resp, err = http.Get(srv.URL + "/api/tickets/" + ticketID)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get ticket: expected 200, got %d", resp.StatusCode)
	}

	var updatedTicket map[string]any
	json.NewDecoder(resp.Body).Decode(&updatedTicket)
	resp.Body.Close()

	if updatedTicket["status"] != "checked_in" {
		t.Fatalf("expected checked_in status, got %s", updatedTicket["status"])
	}
	t.Log("Workflow complete: register → create event → get ticket → check-in")
}

func TestWorkflowDoubleCheckinFails(t *testing.T) {
	srv := httptest.NewServer(workflowServer())
	defer srv.Close()

	// Register
	http.Post(srv.URL+"/api/auth/world-id", "application/json",
		bytes.NewReader([]byte(`{"nullifier_hash":"0xdup","proof":"p"}`)))

	// Create event
	req, _ := http.NewRequest("POST", srv.URL+"/api/events",
		bytes.NewReader([]byte(`{"title":"Dup","capacity":10,"min_age":18}`)))
	req.Header.Set("Authorization", "Bearer jwt-0xdup")
	http.DefaultClient.Do(req)

	// Issue ticket
	req, _ = http.NewRequest("POST", srv.URL+"/api/events/event-1/tickets",
		bytes.NewReader([]byte(`{"zk_proof_commitment":"pc","ticket_type":"standard"}`)))
	req.Header.Set("Authorization", "Bearer jwt-0xdup")
	req.Header.Set("X-User-ID", "user-0xdup")
	http.DefaultClient.Do(req)

	// First check-in succeeds
	resp, _ := http.Post(srv.URL+"/api/checkin", "application/json",
		bytes.NewReader([]byte(`{"qr_secret":"qr-ticket-1","method":"qr"}`)))
	if resp.StatusCode != http.StatusOK {
		t.Fatal("first check-in should succeed")
	}
	resp.Body.Close()

	// Second check-in with same QR fails
	resp, _ = http.Post(srv.URL+"/api/checkin", "application/json",
		bytes.NewReader([]byte(`{"qr_secret":"qr-ticket-1","method":"qr"}`)))
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("double check-in should fail (404), got %d", resp.StatusCode)
	}
	resp.Body.Close()
}
