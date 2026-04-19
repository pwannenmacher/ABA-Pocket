package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"aba-pocket/internal/models"
	"aba-pocket/internal/pdf"
)

func (h *Handler) PDFSymptom(w http.ResponseWriter, r *http.Request) {
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

	card := symptomToCard(symptom)
	data, err := pdf.GenerateSingleCard(card)
	if err != nil {
		http.Error(w, "PDF-Generierung fehlgeschlagen", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "inline; filename=\"symptom-"+strconv.FormatInt(id, 10)+".pdf\"")
	w.Write(data)
}

func (h *Handler) PDFMedication(w http.ResponseWriter, r *http.Request) {
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

	card := medicationToCard(medication)
	data, err := pdf.GenerateSingleCard(card)
	if err != nil {
		http.Error(w, "PDF-Generierung fehlgeschlagen", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "inline; filename=\"medication-"+strconv.FormatInt(id, 10)+".pdf\"")
	w.Write(data)
}

func (h *Handler) PDFAll(w http.ResponseWriter, r *http.Request) {
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

	// Load full data for each card
	var cards []pdf.CardData
	for _, s := range symptoms {
		full, err := h.repos.Symptoms.GetByID(r.Context(), s.ID)
		if err != nil {
			continue
		}
		cards = append(cards, symptomToCard(full))
	}
	for _, m := range medications {
		full, err := h.repos.Medications.GetByID(r.Context(), m.ID)
		if err != nil {
			continue
		}
		cards = append(cards, medicationToCard(full))
	}

	data, err := pdf.GenerateAllCards(cards)
	if err != nil {
		http.Error(w, "PDF-Generierung fehlgeschlagen", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=\"aba-pocket-karten.pdf\"")
	w.Write(data)
}

func symptomToCard(s *models.Symptom) pdf.CardData {
	tables := make([]pdf.SymptomTableData, 0, len(s.Tables))
	for _, t := range s.Tables {
		tables = append(tables, pdf.SymptomTableData{
			Title: t.Title,
			Rows:  t.Rows,
		})
	}
	return pdf.CardData{
		Title:     s.Title,
		CardType:  "symptom",
		Tables:    tables,
		Source:    s.Source,
		UpdatedAt: s.UpdatedAt,
	}
}

func medicationToCard(m *models.Medication) pdf.CardData {
	return pdf.CardData{
		Title:     m.Name,
		CardType:  "medication",
		Entries:   m.Entries,
		Source:    m.Source,
		UpdatedAt: m.UpdatedAt,
	}
}
