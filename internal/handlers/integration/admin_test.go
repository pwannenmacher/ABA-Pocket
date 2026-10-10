package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"aba-pocket/internal/auth"
	"aba-pocket/internal/config"
	"aba-pocket/internal/handlers"
	"aba-pocket/internal/models"
	"aba-pocket/internal/repository"
)

const testSecret = "test-secret-32-chars-long-enough!"

// adminClient spielt Requests gegen den echten Router inkl. Auth- und CSRF-Middleware.
type adminClient struct {
	t       *testing.T
	router  http.Handler
	session string
}

func (c *adminClient) do(method, path string, form url.Values) *httptest.ResponseRecorder {
	c.t.Helper()
	var body *strings.Reader
	if form != nil {
		if c.session != "" {
			form.Set("csrf_token", auth.GenerateCSRFToken(testSecret, c.session))
		}
		body = strings.NewReader(form.Encode())
	} else {
		body = strings.NewReader("")
	}
	r := httptest.NewRequest(method, path, body)
	if form != nil {
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if c.session != "" {
		r.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: c.session})
	}
	w := httptest.NewRecorder()
	c.router.ServeHTTP(w, r)
	return w
}

func (c *adminClient) expect(w *httptest.ResponseRecorder, status int, location string) {
	c.t.Helper()
	if w.Code != status {
		c.t.Fatalf("status = %d, want %d (body %q)", w.Code, status, w.Body.String())
	}
	if location != "" && w.Header().Get("Location") != location {
		c.t.Errorf("Location = %q, want %q", w.Header().Get("Location"), location)
	}
}

func cleanAll(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	for _, tbl := range []string{"symptom_medications", "symptom_table_rows", "symptom_tables", "medication_entries", "sessions", "symptoms", "medications", "users"} {
		if _, err := testPool.Exec(ctx, "DELETE FROM "+tbl); err != nil {
			t.Fatalf("clean %s: %v", tbl, err)
		}
	}
}

func newAdminClient(t *testing.T) (*adminClient, *repository.Repositories) {
	t.Helper()
	cleanAll(t)
	repos := repository.New(testPool)
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret-pass"), bcrypt.MinCost)
	if _, err := repos.Users.Create(context.Background(), "admin", "admin@example.org", string(hash)); err != nil {
		t.Fatalf("create admin: %v", err)
	}
	t.Chdir(filepath.Join("..", "..", ".."))
	h := handlers.New(&config.Config{DevMode: true, SessionSecret: testSecret}, repos)
	return &adminClient{t: t, router: h.Router()}, repos
}

func (c *adminClient) login() {
	c.t.Helper()
	w := c.do(http.MethodPost, "/admin/login", url.Values{"username": {"admin"}, "password": {"secret-pass"}})
	c.expect(w, http.StatusSeeOther, "/admin")
	for _, ck := range w.Result().Cookies() {
		if ck.Name == auth.SessionCookieName {
			c.session = ck.Value
		}
	}
	if c.session == "" {
		c.t.Fatal("no session cookie after login")
	}
}

func TestAdmin_Login(t *testing.T) {
	c, _ := newAdminClient(t)

	c.expect(c.do(http.MethodGet, "/admin/login", nil), http.StatusOK, "")
	c.expect(c.do(http.MethodPost, "/admin/login", url.Values{"username": {"admin"}, "password": {"falsch"}}),
		http.StatusSeeOther, "/admin/login?error=1")
	c.expect(c.do(http.MethodPost, "/admin/login", url.Values{"username": {"nobody"}, "password": {"x"}}),
		http.StatusSeeOther, "/admin/login?error=1")

	c.login()
	c.expect(c.do(http.MethodGet, "/admin/", nil), http.StatusOK, "")

	c.expect(c.do(http.MethodPost, "/admin/logout", url.Values{}), http.StatusSeeOther, "/")
	c.expect(c.do(http.MethodGet, "/admin/", nil), http.StatusSeeOther, "/admin/login")
}

func TestAdmin_SymptomCRUD(t *testing.T) {
	c, repos := newAdminClient(t)
	c.login()
	ctx := context.Background()

	c.expect(c.do(http.MethodGet, "/admin/symptoms", nil), http.StatusOK, "")
	c.expect(c.do(http.MethodGet, "/admin/symptoms/new", nil), http.StatusOK, "")
	c.expect(c.do(http.MethodPost, "/admin/symptoms/new", url.Values{"title": {" "}}), http.StatusUnprocessableEntity, "")

	medID, _ := repos.Medications.Create(ctx, &models.Medication{Name: "Adrenalin"})
	c.expect(c.do(http.MethodPost, "/admin/symptoms/new", url.Values{
		"title":            {"Anaphylaxie"},
		"table_count":      {"1"},
		"table_0_title":    {"Erwachsene"},
		"row_count_0":      {"1"},
		"row_0_0_med":      {"Adrenalin"},
		"row_0_0_right":    {"0,5 mg i.m."},
		"medication_ids[]": {strconv.FormatInt(medID, 10)},
	}), http.StatusSeeOther, "/admin/symptoms")

	list, _ := repos.Symptoms.List(ctx)
	if len(list) != 1 {
		t.Fatalf("want 1 symptom, got %d", len(list))
	}
	id := strconv.FormatInt(list[0].ID, 10)

	c.expect(c.do(http.MethodGet, "/admin/symptoms/"+id+"/edit", nil), http.StatusOK, "")
	c.expect(c.do(http.MethodGet, "/admin/symptoms/abc/edit", nil), http.StatusNotFound, "")
	c.expect(c.do(http.MethodGet, "/admin/symptoms/999999/edit", nil), http.StatusNotFound, "")
	c.expect(c.do(http.MethodPost, "/admin/symptoms/"+id+"/edit", url.Values{"title": {"Anaphylaxie neu"}}),
		http.StatusSeeOther, "/admin/symptoms")
	c.expect(c.do(http.MethodPost, "/admin/symptoms/abc/edit", url.Values{}), http.StatusNotFound, "")

	got, _ := repos.Symptoms.GetByID(ctx, list[0].ID)
	if got.Title != "Anaphylaxie neu" {
		t.Errorf("title = %q after update", got.Title)
	}

	c.expect(c.do(http.MethodPost, "/admin/symptoms/abc/delete", url.Values{}), http.StatusNotFound, "")
	c.expect(c.do(http.MethodPost, "/admin/symptoms/"+id+"/delete", url.Values{}), http.StatusSeeOther, "/admin/symptoms")
	if n, _ := repos.Symptoms.Count(ctx); n != 0 {
		t.Errorf("want 0 symptoms after delete, got %d", n)
	}
}

func TestAdmin_MedicationCRUD(t *testing.T) {
	c, repos := newAdminClient(t)
	c.login()
	ctx := context.Background()

	c.expect(c.do(http.MethodGet, "/admin/medications", nil), http.StatusOK, "")
	c.expect(c.do(http.MethodGet, "/admin/medications/new", nil), http.StatusOK, "")
	c.expect(c.do(http.MethodPost, "/admin/medications/new", url.Values{"name": {""}}), http.StatusUnprocessableEntity, "")

	symID, _ := repos.Symptoms.Create(ctx, &models.Symptom{Title: "Anaphylaxie"})
	c.expect(c.do(http.MethodPost, "/admin/medications/new", url.Values{
		"name":          {"Adrenalin"},
		"entry_left[]":  {"Dosis"},
		"entry_right[]": {"0,5 mg"},
		"symptom_ids[]": {strconv.FormatInt(symID, 10)},
	}), http.StatusSeeOther, "/admin/medications")

	list, _ := repos.Medications.List(ctx)
	if len(list) != 1 {
		t.Fatalf("want 1 medication, got %d", len(list))
	}
	id := strconv.FormatInt(list[0].ID, 10)

	c.expect(c.do(http.MethodGet, "/admin/medications/"+id+"/edit", nil), http.StatusOK, "")
	c.expect(c.do(http.MethodGet, "/admin/medications/abc/edit", nil), http.StatusNotFound, "")
	c.expect(c.do(http.MethodGet, "/admin/medications/999999/edit", nil), http.StatusNotFound, "")
	c.expect(c.do(http.MethodPost, "/admin/medications/"+id+"/edit", url.Values{
		"name":         {"Adrenalin 1 mg/ml"},
		"entry_left[]": {"Dosis"},
	}), http.StatusSeeOther, "/admin/medications")
	c.expect(c.do(http.MethodPost, "/admin/medications/abc/edit", url.Values{}), http.StatusNotFound, "")

	linked, _ := repos.Medications.GetLinkedSymptomIDs(ctx, list[0].ID)
	if len(linked) != 0 {
		t.Errorf("symptom links should be cleared by update, got %v", linked)
	}

	c.expect(c.do(http.MethodPost, "/admin/medications/abc/delete", url.Values{}), http.StatusNotFound, "")
	c.expect(c.do(http.MethodPost, "/admin/medications/"+id+"/delete", url.Values{}), http.StatusSeeOther, "/admin/medications")
}

func TestAdmin_Users(t *testing.T) {
	c, repos := newAdminClient(t)
	c.login()
	ctx := context.Background()

	c.expect(c.do(http.MethodGet, "/admin/users", nil), http.StatusOK, "")
	for _, form := range []url.Values{
		{"username": {""}, "password": {"x"}},                // Pflichtfelder
		{"username": {"kurz"}, "password": {"1234567"}},      // Passwort zu kurz
		{"username": {"admin"}, "password": {"lang-genug!"}}, // existiert bereits
		{"username": {"neu"}, "password": {"lang-genug!"}},   // ok
	} {
		c.expect(c.do(http.MethodPost, "/admin/users/new", form), http.StatusSeeOther, "/admin/users")
	}

	users, _ := repos.Users.List(ctx)
	if len(users) != 2 {
		t.Fatalf("want 2 users, got %d", len(users))
	}
	var self, other int64
	for _, u := range users {
		if u.Username == "admin" {
			self = u.ID
		} else {
			other = u.ID
		}
	}

	c.expect(c.do(http.MethodPost, "/admin/users/abc/delete", url.Values{}), http.StatusNotFound, "")
	c.expect(c.do(http.MethodPost, "/admin/users/"+strconv.FormatInt(self, 10)+"/delete", url.Values{}),
		http.StatusSeeOther, "/admin/users")
	c.expect(c.do(http.MethodPost, "/admin/users/"+strconv.FormatInt(other, 10)+"/delete", url.Values{}),
		http.StatusSeeOther, "/admin/users")

	if n, _ := repos.Users.Count(ctx); n != 1 {
		t.Errorf("want 1 user left (self-delete blocked), got %d", n)
	}
}

// TestAdmin_DatabaseErrors ruft die Handler direkt mit einem geschlossenen Pool auf,
// damit jede DB-Operation fehlschlägt und die Fehlerpfade greifen.
func TestAdmin_DatabaseErrors(t *testing.T) {
	cfg := testPool.Config().Copy()
	closed, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	closed.Close()

	t.Chdir(filepath.Join("..", "..", ".."))
	h := handlers.New(&config.Config{DevMode: true, SessionSecret: testSecret}, repository.New(closed))

	call := func(fn http.HandlerFunc, method, id string, form url.Values) int {
		t.Helper()
		r := httptest.NewRequest(method, "/", strings.NewReader(form.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", id)
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
		w := httptest.NewRecorder()
		fn(w, r)
		return w.Code
	}

	tests := []struct {
		name string
		fn   http.HandlerFunc
		form url.Values
		want int
	}{
		{"login", h.AdminLoginPost, url.Values{"username": {"admin"}}, http.StatusSeeOther},
		{"dashboard", h.AdminDashboard, nil, http.StatusOK},
		{"list symptoms", h.AdminListSymptoms, nil, http.StatusInternalServerError},
		{"new symptom", h.AdminNewSymptom, nil, http.StatusOK},
		{"create symptom invalid", h.AdminCreateSymptom, url.Values{}, http.StatusUnprocessableEntity},
		{"create symptom", h.AdminCreateSymptom, url.Values{"title": {"X"}}, http.StatusInternalServerError},
		{"edit symptom", h.AdminEditSymptom, nil, http.StatusNotFound},
		{"update symptom", h.AdminUpdateSymptom, url.Values{"title": {"X"}}, http.StatusInternalServerError},
		{"delete symptom", h.AdminDeleteSymptom, nil, http.StatusInternalServerError},
		{"list medications", h.AdminListMedications, nil, http.StatusInternalServerError},
		{"new medication", h.AdminNewMedication, nil, http.StatusOK},
		{"create medication invalid", h.AdminCreateMedication, url.Values{}, http.StatusUnprocessableEntity},
		{"create medication", h.AdminCreateMedication, url.Values{"name": {"X"}}, http.StatusInternalServerError},
		{"edit medication", h.AdminEditMedication, nil, http.StatusNotFound},
		{"update medication", h.AdminUpdateMedication, url.Values{"name": {"X"}}, http.StatusInternalServerError},
		{"delete medication", h.AdminDeleteMedication, nil, http.StatusInternalServerError},
		{"list users", h.AdminListUsers, nil, http.StatusInternalServerError},
		{"create user", h.AdminCreateUser, url.Values{"username": {"u"}, "password": {"lang-genug!"}}, http.StatusSeeOther},
		{"delete user", h.AdminDeleteUser, nil, http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.form == nil {
				tt.form = url.Values{}
			}
			if got := call(tt.fn, http.MethodPost, "1", tt.form); got != tt.want {
				t.Errorf("status = %d, want %d", got, tt.want)
			}
		})
	}
}
