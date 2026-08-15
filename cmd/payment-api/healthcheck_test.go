package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRunStandaloneHealthcheckSkipsOtherCommands(t *testing.T) {
	handled, err := runStandaloneHealthcheck([]string{"serve"})
	if err != nil || handled {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
}

func TestRunStandaloneHealthcheckCallsEndpointWithoutPocketBase(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	handled, err := runStandaloneHealthcheck([]string{"healthcheck", "--url", server.URL})
	if err != nil || !handled {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
	if calls != 1 {
		t.Fatalf("calls=%d", calls)
	}
}

func TestRunStandaloneHealthcheckFailsOnUnhealthyStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	handled, err := runStandaloneHealthcheck([]string{"healthcheck", "--url=" + server.URL})
	if !handled || err == nil {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
}
