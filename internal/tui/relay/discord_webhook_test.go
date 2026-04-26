package relay

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDiscordRelay_Send_PostsCorrectPayload(t *testing.T) {
	var received struct {
		Content string `json:"content"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		w.WriteHeader(http.StatusNoContent) // Discord returns 204 on success
	}))
	defer srv.Close()

	relay := NewDiscordRelay(srv.URL)
	if err := relay.Send("hello from test"); err != nil {
		t.Fatalf("Send returned error: %v", err)
	}

	if received.Content != "hello from test" {
		t.Errorf("expected content %q, got %q", "hello from test", received.Content)
	}
}

func TestDiscordRelay_Send_EmptyMessage_NoHTTPRequest(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	relay := NewDiscordRelay(srv.URL)
	if err := relay.Send(""); err != nil {
		t.Fatalf("expected no error for empty message, got %v", err)
	}
	if called {
		t.Error("expected no HTTP request for empty message")
	}
}

func TestDiscordRelay_Send_HTTPErrorStatus_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	relay := NewDiscordRelay(srv.URL)
	err := relay.Send("test message")
	if err == nil {
		t.Fatal("expected error on HTTP 429 response")
	}
}

func TestDiscordRelay_Send_RateLimitEnforced(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	const shortInterval = 50 * time.Millisecond

	relay := &DiscordRelay{
		webhookURL:   srv.URL,
		client:       &http.Client{Timeout: 5 * time.Second},
		rateInterval: shortInterval,
		lastSent:     time.Now(), // simulate a very recent prior send
	}

	start := time.Now()
	if err := relay.Send("delayed message"); err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	elapsed := time.Since(start)

	if elapsed < shortInterval-5*time.Millisecond {
		t.Errorf("rate limit not enforced: elapsed %v < interval %v", elapsed, shortInterval)
	}
	if callCount != 1 {
		t.Errorf("expected 1 HTTP call, got %d", callCount)
	}
}

func TestDiscordRelay_Send_NoRateLimitWhenCoolEnough(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	relay := &DiscordRelay{
		webhookURL:   srv.URL,
		client:       &http.Client{Timeout: 5 * time.Second},
		rateInterval: 2 * time.Second,
		lastSent:     time.Now().Add(-5 * time.Second), // long ago
	}

	start := time.Now()
	if err := relay.Send("immediate message"); err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	elapsed := time.Since(start)

	// Should not wait at all
	if elapsed > 500*time.Millisecond {
		t.Errorf("unexpected delay when last send was old: elapsed %v", elapsed)
	}
}
