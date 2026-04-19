package pdf

import (
	"testing"
	"time"

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
