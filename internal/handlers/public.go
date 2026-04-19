package handlers

import (
	"log"
	"net/http"
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
