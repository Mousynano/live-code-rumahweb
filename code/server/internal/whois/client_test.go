package whois

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCheckMapsOfficialAvailabilityField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("domain"); got != "example.com" {
			t.Fatalf("unexpected domain: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"domain":"example.com","availability":"UNAVAILABLE"}`))
	}))
	defer server.Close()

	client := New(server.URL, "secret", time.Second, 0)
	status, checkError := client.Check(context.Background(), "example.com")
	if checkError != nil {
		t.Fatalf("unexpected error: %s", *checkError)
	}
	if status != Taken {
		t.Fatalf("expected TAKEN, got %s", status)
	}
}

func TestCheckMapsAvailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"domain":"available-example.test","availability":"AVAILABLE"}`))
	}))
	defer server.Close()

	client := New(server.URL, "secret", time.Second, 0)
	status, checkError := client.Check(context.Background(), "available-example.test")
	if checkError != nil {
		t.Fatalf("unexpected error: %s", *checkError)
	}
	if status != Available {
		t.Fatalf("expected AVAILABLE, got %s", status)
	}
}
