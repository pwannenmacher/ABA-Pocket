package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"aba-pocket/internal/config"
	"aba-pocket/internal/handlers"
	"aba-pocket/internal/models"
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

// seedSearch legt ein Leitsymptom und ein Medikament an, die auf "asth" bzw. "salb" passen.
func seedSearch(t *testing.T) *handlers.Handler {
	t.Helper()
	ctx := context.Background()
	for _, tbl := range []string{"symptom_medications", "symptom_table_rows", "symptom_tables", "medication_entries", "symptoms", "medications"} {
		if _, err := testPool.Exec(ctx, "DELETE FROM "+tbl); err != nil {
			t.Fatalf("clean %s: %v", tbl, err)
		}
	}
	repos := repository.New(testPool)
	if _, err := repos.Symptoms.Create(ctx, &models.Symptom{Title: "Asthma"}); err != nil {
		t.Fatalf("create symptom: %v", err)
	}
	if _, err := repos.Medications.Create(ctx, &models.Medication{Name: "Salbutamol"}); err != nil {
		t.Fatalf("create medication: %v", err)
	}

	// Templates werden relativ zum Repo-Root geladen.
	t.Chdir(filepath.Join("..", "..", ".."))
	return handlers.New(&config.Config{DevMode: true}, repos)
}

func TestSearch_HtmxDropdown(t *testing.T) {
	h := seedSearch(t)

	r := httptest.NewRequest(http.MethodGet, "/search?q=asthma", nil)
	r.Header.Set("HX-Request", "true")
	r.Header.Set("HX-Target", "div#search-results-dropdown")
	w := httptest.NewRecorder()
	h.Search(w, r)

	body := w.Body.String()
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(body, "dropdown-item") || !strings.Contains(body, "Asthma") {
		t.Errorf("want dropdown fragment with Asthma, got %q", body)
	}
	if strings.Contains(body, "<html") {
		t.Errorf("htmx response must be a fragment, got full page")
	}
}

func TestSearch_FullPage(t *testing.T) {
	h := seedSearch(t)

	r := httptest.NewRequest(http.MethodGet, "/search?q=salbutamol", nil)
	w := httptest.NewRecorder()
	h.Search(w, r)

	body := w.Body.String()
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(body, "<html") || !strings.Contains(body, "Salbutamol") {
		t.Errorf("want full page with Salbutamol, got %q", body)
	}
}
