package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRealIPMiddleware_Trusted(t *testing.T) {
	mw := RealIPMiddleware([]string{"172.18.0.0/16"})
	var got string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.RemoteAddr
	}))
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "172.18.0.9:45564"
	r.Header.Set("X-Forwarded-For", "203.0.113.50, 172.18.0.9")
	handler.ServeHTTP(httptest.NewRecorder(), r)
	if got != "203.0.113.50" {
		t.Errorf("expected 203.0.113.50, got %s", got)
	}
}
func TestRealIPMiddleware_Untrusted(t *testing.T) {
	mw := RealIPMiddleware([]string{"10.0.0.0/8"})
	var got string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.RemoteAddr
	}))
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "172.18.0.9:45564"
	r.Header.Set("X-Forwarded-For", "203.0.113.50")
	handler.ServeHTTP(httptest.NewRecorder(), r)
	if got != "172.18.0.9:45564" {
		t.Errorf("expected unchanged RemoteAddr, got %s", got)
	}
}
func TestRealIPMiddleware_NilProxies(t *testing.T) {
	mw := RealIPMiddleware(nil)
	var got string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.RemoteAddr
	}))
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "172.18.0.9:45564"
	r.Header.Set("X-Forwarded-For", "203.0.113.50")
	handler.ServeHTTP(httptest.NewRecorder(), r)
	if got != "172.18.0.9:45564" {
		t.Errorf("expected unchanged RemoteAddr, got %s", got)
	}
}

func TestRealIPMiddleware_XRealIP(t *testing.T) {
	// X-Real-Ip header should be used as fallback when X-Forwarded-For is absent
	mw := RealIPMiddleware([]string{"172.18.0.0/16"})
	var got string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.RemoteAddr
	}))
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "172.18.0.9:45564"
	r.Header.Set("X-Real-Ip", "203.0.113.100")
	handler.ServeHTTP(httptest.NewRecorder(), r)
	if got != "203.0.113.100" {
		t.Errorf("expected 203.0.113.100 from X-Real-Ip, got %s", got)
	}
}

func TestRealIPMiddleware_PlainIPProxy(t *testing.T) {
	// A plain IP (no CIDR notation) should be treated as /32 and match exactly
	mw := RealIPMiddleware([]string{"172.18.0.9"})
	var got string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.RemoteAddr
	}))
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "172.18.0.9:45564"
	r.Header.Set("X-Forwarded-For", "203.0.113.50")
	handler.ServeHTTP(httptest.NewRecorder(), r)
	if got != "203.0.113.50" {
		t.Errorf("expected 203.0.113.50, got %s", got)
	}
}

func TestRealIPMiddleware_MultipleForwardedFor(t *testing.T) {
	// Should take the leftmost (original client) IP from the chain
	mw := RealIPMiddleware([]string{"172.18.0.0/16"})
	var got string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.RemoteAddr
	}))
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "172.18.0.9:45564"
	r.Header.Set("X-Forwarded-For", "203.0.113.50, 10.0.0.1, 172.18.0.9")
	handler.ServeHTTP(httptest.NewRecorder(), r)
	if got != "203.0.113.50" {
		t.Errorf("expected leftmost IP 203.0.113.50, got %s", got)
	}
}
