package pdf

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-pdf/fpdf"

	"aba-pocket/internal/models"
)

func TestStripMarkdown(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"**bold**", "bold"},
		{"*italic*", "italic"},
		{"**bold** and *italic*", "bold and italic"},
		{"- item", "\u2022 item"},
		{"plain text", "plain text"},
		{"", ""},
		{"**nested *both***", "nested both"},
	}
	for _, tt := range tests {
		got := stripMarkdown(tt.in)
		if got != tt.want {
			t.Errorf("stripMarkdown(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		in   string
		n    int
		want string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello!", 5, "hell\u2026"},
		{"", 5, ""},
		{"ab", 1, "\u2026"},
		{"\u00e4\u00f6\u00fc\u00df", 3, "\u00e4\u00f6\u2026"},
	}
	for _, tt := range tests {
		got := truncate(tt.in, tt.n)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.in, tt.n, got, tt.want)
		}
	}
}

func TestGenerateSingleCardSymptom(t *testing.T) {
	card := CardData{
		Title:    "Anaphylaxie",
		CardType: "symptom",
		Tables: []SymptomTableData{
			{
				Title: "Erstlinientherapie",
				Rows: []models.SymptomTableRow{
					{Medication: "Adrenalin", RightCol: "0,5 mg i.m."},
				},
			},
		},
		Source:    "AWMF 2023",
		UpdatedAt: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
	}
	data, err := GenerateSingleCard(card)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty PDF data")
	}
	if string(data[:5]) != "%PDF-" {
		t.Error("output should start with PDF header")
	}
}

func TestGenerateSingleCardMedication(t *testing.T) {
	card := CardData{
		Title:    "Adrenalin",
		CardType: "medication",
		Entries: []models.CardEntry{
			{LeftCol: "Wirkstoff", RightCol: "Epinephrin"},
			{LeftCol: "Dosierung", RightCol: "0,5 mg i.m."},
		},
		UpdatedAt: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
	}
	data, err := GenerateSingleCard(card)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data[:5]) != "%PDF-" {
		t.Error("output should start with PDF header")
	}
}

func TestGenerateAllCards(t *testing.T) {
	cards := make([]CardData, 10)
	for i := range cards {
		cards[i] = CardData{
			Title:     "Card",
			CardType:  "symptom",
			UpdatedAt: time.Now(),
		}
	}
	data, err := GenerateAllCards(cards)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty PDF")
	}
}

func TestGenerateAllCardsEmpty(t *testing.T) {
	data, err := GenerateAllCards(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected valid PDF even with no cards")
	}
}

func TestCalcRowH_PositiveHeight(t *testing.T) {
	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.AddPage()

	h := calcRowH(pdf, "Adrenalin", "0,5 mg i.m.", 20.0, 30.0, false)
	if h <= 0 {
		t.Errorf("expected positive height, got %f", h)
	}
}

func TestCalcRowH_LongerTextIncreasesHeight(t *testing.T) {
	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.AddPage()

	shortH := calcRowH(pdf, "A", "B", 20.0, 30.0, false)
	// Long text in a narrow column forces wrapping → more lines → taller row
	longText := strings.Repeat("LangerText ", 15)
	longH := calcRowH(pdf, longText, "B", 10.0, 15.0, false)
	if longH <= shortH {
		t.Errorf("long text in narrow column should produce taller row: short=%f, long=%f", shortH, longH)
	}
}

func TestCalcRowH_RightBold(t *testing.T) {
	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.AddPage()

	// Both rightBold=false and rightBold=true should return a valid positive height
	h1 := calcRowH(pdf, "Med", "Dosierung", 20.0, 30.0, false)
	h2 := calcRowH(pdf, "Med", "Dosierung", 20.0, 30.0, true)
	if h1 <= 0 || h2 <= 0 {
		t.Errorf("expected positive heights, got regular=%f bold=%f", h1, h2)
	}
}

func TestGenerateSingleCard_WithDescription(t *testing.T) {
	card := CardData{
		Title:       "Anaphylaxie",
		Description: "Schwere allergische Reaktion",
		CardType:    "symptom",
		Tables: []SymptomTableData{
			{
				Title: "Erstlinientherapie",
				Rows:  []models.SymptomTableRow{{Medication: "Adrenalin", RightCol: "0,5 mg i.m."}},
			},
		},
		Source:    "AWMF 2023",
		UpdatedAt: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
	}
	data, err := GenerateSingleCard(card)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data[:5]) != "%PDF-" {
		t.Error("output should start with PDF header")
	}
}

func TestGenerateSingleCard_EmptyTables(t *testing.T) {
	// A symptom card with no tables should render the "Keine Einträge" placeholder
	card := CardData{
		Title:     "Empty Symptom",
		CardType:  "symptom",
		Tables:    nil,
		UpdatedAt: time.Now(),
	}
	data, err := GenerateSingleCard(card)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data[:5]) != "%PDF-" {
		t.Error("output should be valid PDF even without tables")
	}
}

func TestGenerateSingleCard_EmptyEntries(t *testing.T) {
	// A medication card with no entries should render the "Keine Einträge" placeholder
	card := CardData{
		Title:     "Empty Medication",
		CardType:  "medication",
		Entries:   nil,
		UpdatedAt: time.Now(),
	}
	data, err := GenerateSingleCard(card)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data[:5]) != "%PDF-" {
		t.Error("output should be valid PDF even without entries")
	}
}

func TestGenerateSingleCard_LongTitle(t *testing.T) {
	// Title exceeding 45 runes should be truncated without error
	card := CardData{
		Title:     strings.Repeat("Anaphylaxie durch Insektenstiche ", 3),
		CardType:  "symptom",
		UpdatedAt: time.Now(),
	}
	data, err := GenerateSingleCard(card)
	if err != nil {
		t.Fatalf("unexpected error with long title: %v", err)
	}
	if string(data[:5]) != "%PDF-" {
		t.Error("output should start with PDF header")
	}
}

func TestGenerateAllCards_PageBoundary(t *testing.T) {
	// 9 cards = first page full (8) + 1 on second page
	cards := make([]CardData, 9)
	for i := range cards {
		cards[i] = CardData{
			Title:     fmt.Sprintf("Karte %d", i+1),
			CardType:  "symptom",
			UpdatedAt: time.Now(),
		}
	}
	data, err := GenerateAllCards(cards)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty PDF for 9 cards")
	}
}

func TestGenerateAllCards_ExactOnePage(t *testing.T) {
	// Exactly 8 cards = exactly one full page
	cards := make([]CardData, 8)
	for i := range cards {
		cards[i] = CardData{
			Title:     fmt.Sprintf("Karte %d", i+1),
			CardType:  "medication",
			Entries:   []models.CardEntry{{LeftCol: "Wirkstoff", RightCol: "Epinephrin"}},
			UpdatedAt: time.Now(),
		}
	}
	data, err := GenerateAllCards(cards)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data[:5]) != "%PDF-" {
		t.Error("output should start with PDF header")
	}
}

func TestGenerateSingleCard_MultipleTables(t *testing.T) {
	card := CardData{
		Title:    "Anaphylaxie",
		CardType: "symptom",
		Tables: []SymptomTableData{
			{
				Title: "Erstlinientherapie",
				Rows: []models.SymptomTableRow{
					{Medication: "Adrenalin", RightCol: "0,5 mg i.m."},
					{Medication: "Volumen", RightCol: "500 ml NaCl 0,9%"},
				},
			},
			{
				Title: "Zweitlinientherapie",
				Rows: []models.SymptomTableRow{
					{Medication: "Prednisolon", RightCol: "250 mg i.v."},
				},
			},
		},
		Source:    "AWMF 2023",
		UpdatedAt: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
	}
	data, err := GenerateSingleCard(card)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data[:5]) != "%PDF-" {
		t.Error("output should start with PDF header")
	}
}
