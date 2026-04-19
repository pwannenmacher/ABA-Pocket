package auth

import (
	"aba-pocket/internal/models"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGenerateSessionID(t *testing.T) {
	id, err := GenerateSessionID()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(id) != 64 {
		t.Errorf("expected 64 hex chars, got %d", len(id))
	}

	id2, _ := GenerateSessionID()
	if id == id2 {
		t.Error("two generated IDs should not be equal")
	}
}

func TestSetSessionCookie(t *testing.T) {
	w := httptest.NewRecorder()
	SetSessionCookie(w, "test-session-id")

	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}

	c := cookies[0]
	if c.Name != SessionCookieName {
		t.Errorf("cookie name = %q, want %q", c.Name, SessionCookieName)
	}
	if c.Value != "test-session-id" {
		t.Errorf("cookie value = %q, want %q", c.Value, "test-session-id")
	}
	if !c.HttpOnly {
		t.Error("cookie should be HttpOnly")
	}
	if !c.Secure {
		t.Error("cookie should be Secure")
	}
	if c.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite = %v, want Lax", c.SameSite)
	}
	if c.MaxAge != int(SessionDuration.Seconds()) {
		t.Errorf("MaxAge = %d, want %d", c.MaxAge, int(SessionDuration.Seconds()))
	}
	if c.Path != "/" {
		t.Errorf("Path = %q, want %q", c.Path, "/")
	}
}

func TestClearSessionCookie(t *testing.T) {
	w := httptest.NewRecorder()
	ClearSessionCookie(w)

	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}

	c := cookies[0]
	if c.Name != SessionCookieName {
		t.Errorf("cookie name = %q, want %q", c.Name, SessionCookieName)
	}
	if c.Value != "" {
		t.Errorf("cookie value = %q, want empty", c.Value)
	}
	if c.MaxAge != -1 {
		t.Errorf("MaxAge = %d, want -1", c.MaxAge)
	}
}

func TestUserFromContext(t *testing.T) {
	t.Run("empty context returns nil", func(t *testing.T) {
		u := UserFromContext(context.Background())
		if u != nil {
			t.Errorf("expected nil, got %v", u)
		}
	})

	t.Run("wrong type returns nil", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), contextKeyUser, "not-a-user")
		u := UserFromContext(ctx)
		if u != nil {
			t.Errorf("expected nil, got %v", u)
		}
	})

	t.Run("returns stored user", func(t *testing.T) {
		want := &models.User{ID: 42, Username: "admin"}
		ctx := context.WithValue(context.Background(), contextKeyUser, want)
		got := UserFromContext(ctx)
		if got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	})
}
