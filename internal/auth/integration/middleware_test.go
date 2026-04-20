package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"aba-pocket/internal/auth"
	"aba-pocket/internal/repository"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx := context.Background()

	pgC, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("test_aba"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		panic("start postgres container: " + err.Error())
	}

	connStr, err := pgC.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic("get connection string: " + err.Error())
	}

	testPool, err = pgxpool.New(ctx, connStr)
	if err != nil {
		panic("connect to test db: " + err.Error())
	}

	migrationPath := filepath.Join("..", "..", "..", "migrations", "001_schema.sql")
	data, err := os.ReadFile(migrationPath)
	if err != nil {
		panic("read migration: " + err.Error())
	}
	if _, err := testPool.Exec(ctx, string(data)); err != nil {
		panic("run migration: " + err.Error())
	}

	code := m.Run()
	testPool.Close()
	_ = pgC.Terminate(ctx)
	os.Exit(code)
}

func repos() *repository.Repositories {
	return repository.New(testPool)
}

func cleanDB(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	tables := []string{"symptom_medications", "symptom_table_rows", "symptom_tables", "medication_entries", "sessions", "symptoms", "medications", "users"}
	for _, tbl := range tables {
		if _, err := testPool.Exec(ctx, "DELETE FROM "+tbl); err != nil {
			t.Fatalf("clean %s: %v", tbl, err)
		}
	}
}

func TestAuthMiddleware_NoCookie(t *testing.T) {
	cleanDB(t)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not be called without cookie")
	})
	handler := auth.Middleware(repos(), false)(inner)

	r := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected 303, got %d", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/admin/login" {
		t.Errorf("redirect to %q, want /admin/login", loc)
	}
}

func TestAuthMiddleware_InvalidSession(t *testing.T) {
	cleanDB(t)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not be called")
	})
	handler := auth.Middleware(repos(), false)(inner)

	r := httptest.NewRequest(http.MethodGet, "/admin", nil)
	r.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "nonexistent"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected redirect, got %d", w.Code)
	}
}

func TestAuthMiddleware_ValidSession(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	rp := repos()

	u, _ := rp.Users.Create(ctx, "admin", "", "$2a$10$dummy")
	rp.Users.CreateSession(ctx, "valid-sess", u.ID, 24*time.Hour)

	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		user := auth.UserFromContext(r.Context())
		if user == nil {
			t.Error("user should be in context")
		} else if user.Username != "admin" {
			t.Errorf("username = %q", user.Username)
		}
	})
	handler := auth.Middleware(rp, false)(inner)

	r := httptest.NewRequest(http.MethodGet, "/admin", nil)
	r.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "valid-sess"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if !called {
		t.Error("inner handler should be called")
	}
}

func TestAuthMiddleware_ExpiredSession(t *testing.T) {
	cleanDB(t)
	ctx := context.Background()
	rp := repos()

	u, _ := rp.Users.Create(ctx, "admin", "", "$2a$10$dummy")
	rp.Users.CreateSession(ctx, "expired-sess", u.ID, -1*time.Hour)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not be called")
	})
	handler := auth.Middleware(rp, false)(inner)

	r := httptest.NewRequest(http.MethodGet, "/admin", nil)
	r.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "expired-sess"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected redirect, got %d", w.Code)
	}
}
