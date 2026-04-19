package pdf

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"

	"aba-pocket/internal/models"
)

// DIN A7 landscape: 105mm × 74mm — 8 cards per A4 portrait (2 cols × 4 rows)
const (
	cardW = 105.0
	cardH = 74.0
	cols  = 2
	rows  = 4
)

type CardData struct {
	Title     string
	CardType  string // "symptom" or "medication"
	Entries   []models.CardEntry
	Source    string
	UpdatedAt time.Time
}

// GenerateAllCards generates an A4 PDF with 8 A7 cards per page.
func GenerateAllCards(cards []CardData) ([]byte, error) {
	pdf := newPDF()

	for i, card := range cards {
		if i%(cols*rows) == 0 {
			pdf.AddPage()
		}
		pos := i % (cols * rows)
		col := pos % cols
		row := pos / cols
		x := float64(col) * cardW
		y := float64(row) * cardH
		renderCard(pdf, card, x, y)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("pdf output: %w", err)
	}
	return buf.Bytes(), nil
}

// GenerateSingleCard generates a single A7-sized PDF for one card.
func GenerateSingleCard(card CardData) ([]byte, error) {
	pdf := fpdf.NewCustom(&fpdf.InitType{
		OrientationStr: "L",
		UnitStr:        "mm",
		SizeStr:        "",
		Size:           fpdf.SizeType{Wd: cardW, Ht: cardH},
		FontDirStr:     "",
	})
	pdf.SetMargins(0, 0, 0)
	pdf.SetAutoPageBreak(false, 0)
	pdf.AddPage()
	renderCard(pdf, card, 0, 0)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("pdf output: %w", err)
	}
	return buf.Bytes(), nil
}

func newPDF() *fpdf.Fpdf {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(0, 0, 0)
	pdf.SetAutoPageBreak(false, 0)
	return pdf
}

var reMarkdownBold = regexp.MustCompile(`\*\*(.+?)\*\*`)
var reMarkdownItalic = regexp.MustCompile(`\*(.+?)\*`)

// stripMarkdown removes markdown syntax for plain PDF text.
func stripMarkdown(s string) string {
	s = reMarkdownBold.ReplaceAllString(s, "$1")
	s = reMarkdownItalic.ReplaceAllString(s, "$1")
	s = strings.ReplaceAll(s, "- ", "• ")
	return s
}

func renderCard(pdf *fpdf.Fpdf, card CardData, x, y float64) {
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	border := 0.3 // mm line width
	pdf.SetLineWidth(border)

	// ── Outer border ──────────────────────────────────────────────────
	pdf.SetDrawColor(100, 100, 100)
	pdf.Rect(x, y, cardW, cardH, "D")

	// ── Title bar ─────────────────────────────────────────────────────
	titleH := 9.0
	if card.CardType == "symptom" {
		pdf.SetFillColor(30, 80, 140) // dark blue for symptoms
	} else {
		pdf.SetFillColor(20, 120, 80) // dark green for medications
	}
	pdf.Rect(x, y, cardW, titleH, "F")

	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetXY(x+2, y+1)
	pdf.CellFormat(cardW-4, titleH-2, tr(truncate(card.Title, 45)), "", 0, "LM", false, 0, "")

	// Type label (top-right corner)
	pdf.SetFont("Helvetica", "", 6)
	label := "Leitsymptom"
	if card.CardType == "medication" {
		label = "Medikament"
	}
	pdf.SetXY(x+2, y+1)
	pdf.CellFormat(cardW-4, titleH-2, tr(label), "", 0, "RM", false, 0, "")

	// ── Entries table ─────────────────────────────────────────────────
	footerH := 5.0
	tableY := y + titleH
	tableH := cardH - titleH - footerH
	leftColW := cardW * 0.40
	rightColW := cardW - leftColW

	pdf.SetDrawColor(180, 180, 180)
	pdf.SetLineWidth(0.2)
	// Vertical divider between columns
	pdf.Line(x+leftColW, tableY, x+leftColW, y+cardH-footerH)

	if len(card.Entries) == 0 {
		pdf.SetTextColor(160, 160, 160)
		pdf.SetFont("Helvetica", "I", 7)
		pdf.SetXY(x+2, tableY+2)
		pdf.CellFormat(cardW-4, 6, tr("Keine Einträge"), "", 0, "LT", false, 0, "")
	} else {
		rowH := tableH / float64(len(card.Entries))
		if rowH > 8 {
			rowH = 8
		}
		if rowH < 4 {
			rowH = 4
		}

		for i, entry := range card.Entries {
			ey := tableY + float64(i)*rowH
			if ey+rowH > y+cardH-footerH {
				break // no more space
			}

			// Row separator
			if i > 0 {
				pdf.Line(x, ey, x+cardW, ey)
			}

			// Left column (key) – bold
			pdf.SetTextColor(40, 40, 40)
			pdf.SetFont("Helvetica", "B", 6.5)
			pdf.SetXY(x+1.5, ey+0.5)
			leftText := tr(stripMarkdown(entry.LeftCol))
			pdf.MultiCell(leftColW-2, rowH*0.45, leftText, "", "LT", false)

			// Right column (value)
			pdf.SetFont("Helvetica", "", 6.5)
			pdf.SetXY(x+leftColW+1.5, ey+0.5)
			rightText := tr(stripMarkdown(entry.RightCol))
			pdf.MultiCell(rightColW-2, rowH*0.45, rightText, "", "LT", false)
		}
	}

	// ── Footer ────────────────────────────────────────────────────────
	footerY := y + cardH - footerH
	pdf.SetFillColor(240, 240, 240)
	pdf.Rect(x, footerY, cardW, footerH, "F")
	pdf.Line(x, footerY, x+cardW, footerY)

	pdf.SetTextColor(100, 100, 100)
	pdf.SetFont("Helvetica", "", 5)
	pdf.SetXY(x+1.5, footerY+0.8)
	sourceText := ""
	if card.Source != "" {
		sourceText = tr(truncate("Quelle: "+card.Source, 40))
	}
	pdf.CellFormat((cardW/2)-2, footerH-1, sourceText, "", 0, "LM", false, 0, "")

	updText := tr("Stand: " + card.UpdatedAt.Format("01/2006"))
	pdf.SetXY(x+cardW/2, footerY+0.8)
	pdf.CellFormat((cardW/2)-2, footerH-1, updText, "", 0, "RM", false, 0, "")
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-1]) + "…"
}
