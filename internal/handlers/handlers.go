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

type Handler struct {
	cfg          *config.Config
	repos        *repository.Repositories
	loginLimiter *auth.LoginLimiter

	tmplMu    sync.RWMutex
	tmplCache map[string]*template.Template
	funcMap   template.FuncMap
}

func New(cfg *config.Config, repos *repository.Repositories) *Handler {
	h := &Handler{
		cfg:          cfg,
		repos:        repos,
		loginLimiter: auth.NewLoginLimiter(5, 15*time.Minute),
		tmplCache:    make(map[string]*template.Template),
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
		"add": func(a, b int) int { return a + b },
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

func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))

	// Static files
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	r.Get("/health", h.Health)

	// Public routes
	r.Get("/", h.Index)
	r.Get("/symptoms", h.ListSymptoms)
	r.Get("/symptoms/{id}", h.GetSymptom)
	r.Get("/medications", h.ListMedications)
	r.Get("/medications/{id}", h.GetMedication)
	r.Get("/search", h.Search)
	r.Get("/disclaimer", h.Disclaimer)
	r.Get("/imprint", h.Imprint)

	// PDF export
	r.Get("/pdf/symptoms/{id}", h.PDFSymptom)
	r.Get("/pdf/medications/{id}", h.PDFMedication)
	r.Get("/pdf/all", h.PDFAll)

	// Admin routes
	r.Get("/admin/login", h.AdminLogin)
	r.Post("/admin/login", h.AdminLoginPost)

	r.Route("/admin", func(r chi.Router) {
		r.Use(auth.Middleware(h.repos, !h.cfg.DevMode))
		r.Use(h.csrfProtect)

		r.Post("/logout", h.AdminLogout)

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
	})

	return r
}

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

func (h *Handler) render(w http.ResponseWriter, status int, page string, data any) {
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

func (h *Handler) renderAdmin(w http.ResponseWriter, r *http.Request, status int, page string, data any) {
	// CSRF-Token automatisch in PageData einfügen
	if pd, ok := data.(PageData); ok && pd.CSRFToken == "" {
		pd.CSRFToken = auth.CSRFTokenFromRequest(r, h.cfg.SessionSecret)
		data = pd
	}
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

func (h *Handler) setFlash(w http.ResponseWriter, msg string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "flash",
		Value:    url.QueryEscape(msg),
		Path:     "/admin",
		MaxAge:   60,
		HttpOnly: true,
		Secure:   !h.cfg.DevMode,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) getFlash(w http.ResponseWriter, r *http.Request) string {
	cookie, err := r.Cookie("flash")
	if err != nil {
		return ""
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "flash",
		Value:    "",
		Path:     "/admin",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   !h.cfg.DevMode,
	})
	msg, _ := url.QueryUnescape(cookie.Value)
	return msg
}

type PageData struct {
	Title     string
	User      *models.User
	Flash     string
	CSRFToken string
	Data      any
}

// csrfProtect validiert CSRF-Tokens bei POST-Requests.
func (h *Handler) csrfProtect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			cookie, err := r.Cookie(auth.SessionCookieName)
			if err != nil {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			token := r.FormValue("csrf_token")
			if !auth.ValidateCSRFToken(h.cfg.SessionSecret, cookie.Value, token) {
				http.Error(w, "Forbidden – ungültiges CSRF-Token", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
