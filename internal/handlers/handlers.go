package handlers

import (
	"html/template"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"

	"aba-pocket/internal/auth"
	"aba-pocket/internal/config"
	"aba-pocket/internal/models"
	"aba-pocket/internal/repository"
)

// Handler holds all dependencies for HTTP handlers.
type Handler struct {
	cfg   *config.Config
	repos *repository.Repositories

	tmplMu    sync.RWMutex
	tmplCache map[string]*template.Template
	funcMap   template.FuncMap
}

func New(cfg *config.Config, repos *repository.Repositories) *Handler {
	h := &Handler{
		cfg:       cfg,
		repos:     repos,
		tmplCache: make(map[string]*template.Template),
	}
	h.funcMap = template.FuncMap{
		"markdown":    renderMarkdown,
		"formatDate":  func(t time.Time) string { return t.Format("02.01.2006") },
		"formatMonth": func(t time.Time) string { return t.Format("01/2006") },
		"truncate": func(s string, n int) string {
			if len([]rune(s)) <= n {
				return s
			}
			return string([]rune(s)[:n]) + "…"
		},
		"safeHTML": func(s string) template.HTML { return template.HTML(s) },
		"add":      func(a, b int) int { return a + b },
	}
	return h
}

func renderMarkdown(s string) template.HTML {
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs |
		parser.NoEmptyLineBeforeBlock | parser.HardLineBreak
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse([]byte(s))

	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	return template.HTML(markdown.Render(doc, renderer))
}

// Router builds and returns the chi router.
func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))

	// Static files
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	// Public routes
	r.Get("/", h.Index)
	r.Get("/symptoms", h.ListSymptoms)
	r.Get("/symptoms/{id}", h.GetSymptom)
	r.Get("/medications", h.ListMedications)
	r.Get("/medications/{id}", h.GetMedication)
	r.Get("/search", h.Search)

	// PDF export
	r.Get("/pdf/symptoms/{id}", h.PDFSymptom)
	r.Get("/pdf/medications/{id}", h.PDFMedication)
	r.Get("/pdf/all", h.PDFAll)

	// Admin routes
	r.Get("/admin/login", h.AdminLogin)
	r.Post("/admin/login", h.AdminLoginPost)
	r.Post("/admin/logout", h.AdminLogout)

	r.Route("/admin", func(r chi.Router) {
		r.Use(auth.Middleware(h.repos))

		r.Get("/", h.AdminDashboard)

		// Symptoms CRUD
		r.Get("/symptoms", h.AdminListSymptoms)
		r.Get("/symptoms/new", h.AdminNewSymptom)
		r.Post("/symptoms/new", h.AdminCreateSymptom)
		r.Get("/symptoms/{id}/edit", h.AdminEditSymptom)
		r.Post("/symptoms/{id}/edit", h.AdminUpdateSymptom)
		r.Post("/symptoms/{id}/delete", h.AdminDeleteSymptom)

		// Medications CRUD
		r.Get("/medications", h.AdminListMedications)
		r.Get("/medications/new", h.AdminNewMedication)
		r.Post("/medications/new", h.AdminCreateMedication)
		r.Get("/medications/{id}/edit", h.AdminEditMedication)
		r.Post("/medications/{id}/edit", h.AdminUpdateMedication)
		r.Post("/medications/{id}/delete", h.AdminDeleteMedication)

		// Users
		r.Get("/users", h.AdminListUsers)
		r.Post("/users/new", h.AdminCreateUser)
		r.Post("/users/{id}/delete", h.AdminDeleteUser)

		// HTMX fragment: new entry row
		r.Get("/entries/row", h.AdminEntryRow)
	})

	return r
}

// ─── Template helpers ──────────────────────────────────────────────────────

func (h *Handler) loadTemplate(cacheKey string, files []string) (*template.Template, error) {
	if !h.cfg.DevMode {
		h.tmplMu.RLock()
		if t, ok := h.tmplCache[cacheKey]; ok {
			h.tmplMu.RUnlock()
			return t, nil
		}
		h.tmplMu.RUnlock()
	}

	t, err := template.New("").Funcs(h.funcMap).ParseFiles(files...)
	if err != nil {
		return nil, err
	}

	if !h.cfg.DevMode {
		h.tmplMu.Lock()
		h.tmplCache[cacheKey] = t
		h.tmplMu.Unlock()
	}
	return t, nil
}

func (h *Handler) getTemplate(name string) (*template.Template, error) {
	return h.loadTemplate(name, []string{
		filepath.Join("web", "templates", "layout.html"),
		filepath.Join("web", "templates", name+".html"),
	})
}

func (h *Handler) getAdminTemplate(name string) (*template.Template, error) {
	return h.loadTemplate("admin/"+name, []string{
		filepath.Join("web", "templates", "admin", "layout.html"),
		filepath.Join("web", "templates", "admin", name+".html"),
	})
}

func (h *Handler) render(w http.ResponseWriter, status int, page string, data interface{}) {
	t, err := h.getTemplate(page)
	if err != nil {
		log.Printf("template parse error (%s): %v", page, err)
		http.Error(w, "Interner Fehler", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := t.ExecuteTemplate(w, "layout", data); err != nil {
		log.Printf("template execute error (%s): %v", page, err)
	}
}

func (h *Handler) renderAdmin(w http.ResponseWriter, status int, page string, data interface{}) {
	t, err := h.getAdminTemplate(page)
	if err != nil {
		log.Printf("admin template parse error (%s): %v", page, err)
		http.Error(w, "Interner Fehler", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := t.ExecuteTemplate(w, "admin_layout", data); err != nil {
		log.Printf("admin template execute error (%s): %v", page, err)
	}
}

// ─── Flash messages ────────────────────────────────────────────────────────

func setFlash(w http.ResponseWriter, msg string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "flash",
		Value:    url.QueryEscape(msg),
		Path:     "/admin",
		MaxAge:   60,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func getFlash(w http.ResponseWriter, r *http.Request) string {
	cookie, err := r.Cookie("flash")
	if err != nil {
		return ""
	}
	http.SetCookie(w, &http.Cookie{
		Name:   "flash",
		Value:  "",
		Path:   "/admin",
		MaxAge: -1,
	})
	msg, _ := url.QueryUnescape(cookie.Value)
	return msg
}

// ─── Shared page data ──────────────────────────────────────────────────────

type PageData struct {
	Title string
	User  *models.User
	Flash string
	Data  interface{}
}
