package handlers

import (
	"log"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	symptoms, err := h.repos.Symptoms.List(r.Context())
	if err != nil {
		http.Error(w, "Fehler beim Laden", http.StatusInternalServerError)
		return
	}
	medications, err := h.repos.Medications.List(r.Context())
	if err != nil {
		http.Error(w, "Fehler beim Laden", http.StatusInternalServerError)
		return
	}

	h.render(w, http.StatusOK, "index", PageData{
		Title: "ABA Pocket – Notfallmedizin",
		Data: map[string]interface{}{
			"Symptoms":    symptoms,
			"Medications": medications,
		},
	})
}

func (h *Handler) ListSymptoms(w http.ResponseWriter, r *http.Request) {
	symptoms, err := h.repos.Symptoms.List(r.Context())
	if err != nil {
		http.Error(w, "Fehler beim Laden", http.StatusInternalServerError)
		return
	}
	h.render(w, http.StatusOK, "symptoms", PageData{
		Title: "Leitsymptome",
		Data:  symptoms,
	})
}

func (h *Handler) GetSymptom(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	symptom, err := h.repos.Symptoms.GetByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	h.render(w, http.StatusOK, "symptom", PageData{
		Title: symptom.Title,
		Data:  symptom,
	})
}

func (h *Handler) ListMedications(w http.ResponseWriter, r *http.Request) {
	medications, err := h.repos.Medications.List(r.Context())
	if err != nil {
		http.Error(w, "Fehler beim Laden", http.StatusInternalServerError)
		return
	}
	h.render(w, http.StatusOK, "medications", PageData{
		Title: "Medikamente",
		Data:  medications,
	})
}

func (h *Handler) GetMedication(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	medication, err := h.repos.Medications.GetByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	h.render(w, http.StatusOK, "medication", PageData{
		Title: medication.Name,
		Data:  medication,
	})
}

func (h *Handler) Disclaimer(w http.ResponseWriter, r *http.Request) {
	h.render(w, http.StatusOK, "disclaimer", PageData{Title: "Haftungsausschluss"})
}

func (h *Handler) Imprint(w http.ResponseWriter, r *http.Request) {
	h.render(w, http.StatusOK, "imprint", PageData{
		Title: "Impressum & Datenschutz",
		Data: map[string]string{
			"Name":   h.cfg.ImprintName,
			"Street": h.cfg.ImprintStreet,
			"Zip":    h.cfg.ImprintZip,
			"City":   h.cfg.ImprintCity,
			"Email":  h.cfg.ImprintEmail,
		},
	})
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")

	// HTMX partial request → return only the results fragment
	isHtmx := r.Header.Get("HX-Request") == "true"

	if q == "" {
		if isHtmx {
			return
		}
		h.render(w, http.StatusOK, "search", PageData{
			Title: "Suche",
			Data: map[string]interface{}{
				"Query":       "",
				"Symptoms":    nil,
				"Medications": nil,
			},
		})
		return
	}

	symptoms, err := h.repos.Symptoms.Search(r.Context(), q)
	if err != nil {
		symptoms = nil
	}
	medications, err := h.repos.Medications.Search(r.Context(), q)
	if err != nil {
		medications = nil
	}

	data := map[string]interface{}{
		"Query":       q,
		"Symptoms":    symptoms,
		"Medications": medications,
	}

	if isHtmx {
		// Navbar-Dropdown vs. Suchseiten-Ergebnisse unterscheiden
		target := r.Header.Get("HX-Target")
		if target == "search-results-dropdown" {
			t, err := h.loadTemplate("search_results_dropdown", []string{
				filepath.Join("web", "templates", "search_results.html"),
			})
			if err != nil {
				http.Error(w, "Fehler", http.StatusInternalServerError)
				return
			}
			if err := t.ExecuteTemplate(w, "search_results", data); err != nil {
				log.Printf("search_results dropdown template error: %v", err)
			}
			return
		}
		t, err := h.getTemplate("search")
		if err != nil {
			http.Error(w, "Fehler", http.StatusInternalServerError)
			return
		}
		if err := t.ExecuteTemplate(w, "search_results", data); err != nil {
			log.Printf("search_results template error: %v", err)
		}
		return
	}

	h.render(w, http.StatusOK, "search", PageData{
		Title: "Suche: " + q,
		Data:  data,
	})
}
