package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aba-pocket/internal/config"
	"aba-pocket/internal/models"
)

func searchTestData() map[string]any {
	return map[string]any{
		"Query":       "asth",
		"Symptoms":    []*models.Symptom{{ID: 7, Title: "Asthma", UpdatedAt: time.Now()}},
		"Medications": []*models.Medication{{ID: 3, Name: "Salbutamol", UpdatedAt: time.Now()}},
	}
}

func searchRequest(target string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/search?q=asth", nil)
	r.Header.Set("HX-Request", "true")
	r.Header.Set("HX-Target", target)
	return r
}

func TestRenderSearchFragment(t *testing.T) {
	// Templates werden relativ zum Repo-Root geladen.
	t.Chdir("../..")

	tests := []struct {
		name        string
		target      string
		contains    string
		notContains string
	}{
		{"dropdown htmx 4", "div#search-results-dropdown", "dropdown-item", "card-list-item"},
		{"dropdown htmx 2", "search-results-dropdown", "dropdown-item", "card-list-item"},
		{"search page", "div#search-results", "card-list-item", "dropdown-item"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := New(&config.Config{DevMode: true}, nil)
			w := httptest.NewRecorder()

			h.renderSearchFragment(w, searchRequest(tt.target), searchTestData())

			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", w.Code)
			}
			body := w.Body.String()
			if !strings.Contains(body, tt.contains) {
				t.Errorf("body should contain %q, got %q", tt.contains, body)
			}
			if strings.Contains(body, tt.notContains) {
				t.Errorf("body should not contain %q, got %q", tt.notContains, body)
			}
			if !strings.Contains(body, "Asthma") || !strings.Contains(body, "Salbutamol") {
				t.Errorf("body should contain results, got %q", body)
			}
		})
	}
}

func TestRenderSearchFragment_TemplateMissing(t *testing.T) {
	// Ohne Chdir fehlen die Templates → HTML-Fehlerfragment statt Plaintext.
	t.Chdir(t.TempDir())

	tests := []struct {
		target string
		class  string
	}{
		{"div#search-results-dropdown", `class="dropdown-empty"`},
		{"div#search-results", `class="empty-state"`},
	}
	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			h := New(&config.Config{DevMode: true}, nil)
			w := httptest.NewRecorder()

			h.renderSearchFragment(w, searchRequest(tt.target), searchTestData())

			if w.Code != http.StatusInternalServerError {
				t.Errorf("status = %d, want 500", w.Code)
			}
			if body := w.Body.String(); !strings.Contains(body, tt.class) {
				t.Errorf("body = %q, want %s", body, tt.class)
			}
		})
	}
}

func TestSearch_EmptyQueryHtmx(t *testing.T) {
	h := New(&config.Config{DevMode: true}, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/search", nil)
	r.Header.Set("HX-Request", "true")

	h.Search(w, r)

	if w.Code != http.StatusOK || w.Body.Len() != 0 {
		t.Errorf("got status %d body %q, want empty 200", w.Code, w.Body.String())
	}
}

func TestRenderSearchFragment_ExecuteError(t *testing.T) {
	t.Chdir("../..")

	// []int hat kein Feld .ID → Template-Ausführung schlägt fehl und wird geloggt.
	data := map[string]any{"Query": "x", "Symptoms": []int{1}, "Medications": []int{}}
	for _, target := range []string{"div#search-results-dropdown", "div#search-results"} {
		t.Run(target, func(t *testing.T) {
			h := New(&config.Config{DevMode: true}, nil)
			w := httptest.NewRecorder()

			h.renderSearchFragment(w, searchRequest(target), data)

			if strings.Contains(w.Body.String(), "Salbutamol") {
				t.Errorf("unexpected result content: %q", w.Body.String())
			}
		})
	}
}
