package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCSRFTokenRoundtrip(t *testing.T) {
	secret := "test-secret-32-chars-long-enough!"
	sessionID := "session-abc-123"

	token := GenerateCSRFToken(secret, sessionID)
	if len(token) != 32 {
		t.Errorf("token length = %d, want 32", len(token))
	}
	if !ValidateCSRFToken(secret, sessionID, token) {
		t.Error("freshly generated token should be valid")
	}
}

func TestCSRFTokenInvalid(t *testing.T) {
	secret := "test-secret-32-chars-long-enough!"
	if ValidateCSRFToken(secret, "sess", "wrong-token") {
		t.Error("wrong token should not validate")
	}
	if ValidateCSRFToken(secret, "sess", "") {
		t.Error("empty token should not validate")
	}
}

func TestCSRFTokenDifferentInputs(t *testing.T) {
	secret := "test-secret-32-chars-long-enough!"
	t1 := GenerateCSRFToken(secret, "session-1")
	t2 := GenerateCSRFToken(secret, "session-2")
	if t1 == t2 {
		t.Error("different sessions should produce different tokens")
	}
	t3 := GenerateCSRFToken("other-secret-32-chars-long!!!!!!!!", "session-1")
	if t1 == t3 {
		t.Error("different secrets should produce different tokens")
	}
}

func TestCSRFTokenFromRequest(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if token := CSRFTokenFromRequest(r, "secret"); token != "" {
		t.Errorf("expected empty without cookie, got %q", token)
	}

	r.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "sess-123"})
	secret := "secret-32-chars-long-enough!!!!!"
	token := CSRFTokenFromRequest(r, secret)
	if token != GenerateCSRFToken(secret, "sess-123") {
		t.Error("token mismatch")
	}
}
