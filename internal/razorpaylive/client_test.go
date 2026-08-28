package razorpaylive

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientCreatesOrderWithBasicAuthAndRefusesRedirects(t *testing.T) {
	var targetRequests int
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetRequests++
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, password, ok := r.BasicAuth()
		if !ok || user != "rzp_live_key" || password != "secret" {
			t.Fatalf("basic auth user=%q password=%q ok=%v", user, password, ok)
		}
		if r.URL.Path != "/orders" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer server.Close()
	client := NewClient("rzp_live_key", "secret")
	client.BaseURL = server.URL
	_, err := client.CreateOrder(context.Background(), 100, "receipt")
	if err == nil {
		t.Fatal("expected redirect response to be rejected")
	}
	if targetRequests != 0 {
		t.Fatalf("credentials could have been forwarded to redirect target: requests=%d", targetRequests)
	}
}

func TestClientValidatesProviderOrderAmount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"order_test","amount":999,"currency":"INR","status":"created"}`))
	}))
	defer server.Close()
	client := NewClient("rzp_live_key", "secret")
	client.BaseURL = server.URL
	if _, err := client.CreateOrder(context.Background(), 100, "receipt"); err == nil {
		t.Fatal("expected inconsistent amount error")
	}
}
