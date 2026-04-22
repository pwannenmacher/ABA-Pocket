package handlers

import (
	"net/http"
	"strconv"

	"aba-pocket/internal/models"
	"aba-pocket/internal/pdf"
)

func (h *Handler) PDFSymptom(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	symptom, err := h.repos.Symptoms.GetByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	data, err := pdf.GenerateSingleCard(symptomToCard(symptom))
	servePDF(w, data, err, "symptom-"+strconv.FormatInt(id, 10)+".pdf", "inline")
}

func (h *Handler) PDFMedication(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	medication, err := h.repos.Medications.GetByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	data, err := pdf.GenerateSingleCard(medicationToCard(medication))
	servePDF(w, data, err, "medication-"+strconv.FormatInt(id, 10)+".pdf", "inline")
}

// servePDF writes a generated PDF to the response or returns an error.
func servePDF(w http.ResponseWriter, data []byte, err error, filename, disposition string) {
	if err != nil {
		http.Error(w, "PDF-Generierung fehlgeschlagen", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", disposition+"; filename=\""+filename+"\"")
	_, _ = w.Write(data)
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
	servePDF(w, data, err, "aba-pocket-karten.pdf", "attachment")
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
		Title:       s.Title,
		Description: s.Description,
		CardType:    "symptom",
		Tables:      tables,
		Source:      s.Source,
		UpdatedAt:   s.UpdatedAt,
	}
}

func medicationToCard(m *models.Medication) pdf.CardData {
	return pdf.CardData{
		Title:       m.Name,
		Description: m.Description,
		CardType:    "medication",
		Entries:     m.Entries,
		Source:      m.Source,
		UpdatedAt:   m.UpdatedAt,
	}
}
