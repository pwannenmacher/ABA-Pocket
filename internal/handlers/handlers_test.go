package handlers

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"aba-pocket/internal/auth"
	"aba-pocket/internal/config"
)

func newTestHandler() *Handler {
	cfg := &config.Config{
		SessionSecret: "test-secret-32-chars-long-enough!",
		DevMode:       true,
	}
	return &Handler{
		cfg:          cfg,
		loginLimiter: auth.NewLoginLimiter(5, 15*time.Minute),
		tmplCache:    make(map[string]*template.Template),
	}
}

func TestCSRFProtect_BlocksPostWithoutToken(t *testing.T) {
	h := newTestHandler()

	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	handler := h.csrfProtect(inner)

	form := url.Values{"some_field": {"value"}}
	r := httptest.NewRequest(http.MethodPost, "/admin/something", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "sess-123"})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
	if called {
		t.Error("inner handler should not be called")
	}
}

func TestCSRFProtect_AllowsValidToken(t *testing.T) {
	h := newTestHandler()
	sessionID := "sess-123"
	token := auth.GenerateCSRFToken(h.cfg.SessionSecret, sessionID)

	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := h.csrfProtect(inner)

	form := url.Values{"csrf_token": {token}}
	r := httptest.NewRequest(http.MethodPost, "/admin/something", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if !called {
		t.Error("inner handler should be called with valid token")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestCSRFProtect_AllowsGetRequests(t *testing.T) {
	h := newTestHandler()

	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	handler := h.csrfProtect(inner)

	r := httptest.NewRequest(http.MethodGet, "/admin/something", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if !called {
		t.Error("GET requests should pass through without CSRF check")
	}
}

func TestCSRFProtect_BlocksPostWithoutCookie(t *testing.T) {
	h := newTestHandler()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not be called")
	})

	handler := h.csrfProtect(inner)

	form := url.Values{"csrf_token": {"some-token"}}
	r := httptest.NewRequest(http.MethodPost, "/admin/something", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 without session cookie, got %d", w.Code)
	}
}

func TestSetGetFlash_Roundtrip(t *testing.T) {
	h := newTestHandler()

	// Set the flash message
	w := httptest.NewRecorder()
	h.setFlash(w, "Gespeichert!")

	// Extract the flash cookie from the response
	var flashCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "flash" {
			flashCookie = c
			break
		}
	}
	if flashCookie == nil {
		t.Fatal("flash cookie not set by setFlash")
	}
	if flashCookie.MaxAge != 60 {
		t.Errorf("flash cookie MaxAge = %d, want 60", flashCookie.MaxAge)
	}

	// Read the flash message via getFlash
	r := httptest.NewRequest(http.MethodGet, "/admin", nil)
	r.AddCookie(flashCookie)
	w2 := httptest.NewRecorder()

	msg := h.getFlash(w2, r)
	if msg != "Gespeichert!" {
		t.Errorf("getFlash = %q, want %q", msg, "Gespeichert!")
	}

	// Cookie should be cleared (MaxAge=-1) after reading
	cleared := false
	for _, c := range w2.Result().Cookies() {
		if c.Name == "flash" && c.MaxAge == -1 {
			cleared = true
			break
		}
	}
	if !cleared {
		t.Error("flash cookie should be cleared (MaxAge=-1) after reading")
	}
}

func TestGetFlash_NoCookie(t *testing.T) {
	h := newTestHandler()
	r := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()

	msg := h.getFlash(w, r)
	if msg != "" {
		t.Errorf("expected empty flash without cookie, got %q", msg)
	}
}

func TestSetFlash_SpecialCharacters(t *testing.T) {
	h := newTestHandler()

	w := httptest.NewRecorder()
	original := "Fehler: Ungültige Eingabe & <Sonderzeichen>!"
	h.setFlash(w, original)

	var flashCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "flash" {
			flashCookie = c
			break
		}
	}
	if flashCookie == nil {
		t.Fatal("flash cookie not set")
	}

	// Reading back should restore original message
	r := httptest.NewRequest(http.MethodGet, "/admin", nil)
	r.AddCookie(flashCookie)
	msg := h.getFlash(httptest.NewRecorder(), r)
	if msg != original {
		t.Errorf("flash roundtrip: got %q, want %q", msg, original)
	}
}
