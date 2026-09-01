package v4web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerServesDashboardAndPairingFallbackWithSecurityHeaders(t *testing.T) {
	h := Handler()
	for _, target := range []string{"/", "/device/pair/example-token", "/not-a-real-route"} {
		r := httptest.NewRequest(http.MethodGet, target, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `id="root"`) {
			t.Fatalf("%s status=%d body=%q", target, w.Code, w.Body.String())
		}
		if got := w.Header().Get("Content-Security-Policy"); !strings.Contains(got, "frame-ancestors 'none'") {
			t.Fatalf("%s CSP=%q", target, got)
		}
		if got := w.Header().Get("Cache-Control"); got != "no-store" {
			t.Fatalf("%s cache=%q", target, got)
		}
	}
}

func TestHandlerRejectsWrites(t *testing.T) {
	w := httptest.NewRecorder()
	Handler().ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/", strings.NewReader("x")))
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d", w.Code)
	}
}
