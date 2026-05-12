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

// DIN A7 Hochformat: 74 mm × 105 mm
// Sammel-PDF: DIN A4 Querformat (297 × 210 mm), 4 Spalten × 2 Zeilen = 8 Karten
// 4 × 74 = 296 mm  → 0,5 mm Rand links/rechts zum Zentrieren
// 2 × 105 = 210 mm → exakte A4-Höhe im Querformat
const (
	cardW   = 74.0  // A7 Hochformat Breite
	cardH   = 105.0 // A7 Hochformat Höhe
	cols    = 4
	rows    = 2
	marginX = 0.5 // (297 - 4×74) / 2
	font    = "Helvetica"
)

// SymptomTableData ist die PDF-Darstellung einer Tabellengruppe eines Leitsymptoms.
type SymptomTableData struct {
	Title string
	Rows  []models.SymptomTableRow
}

// CardData enthält alle Daten für eine einzelne Taschenkarte.
type CardData struct {
	Title       string
	Description string
	CardType    string
	Tables      []SymptomTableData
	Entries     []models.CardEntry
	Source      string
	UpdatedAt   time.Time
}

// GenerateAllCards erzeugt ein A4-Querformat-PDF mit 8 A7-Hochformat-Karten pro Seite.
func GenerateAllCards(cards []CardData) ([]byte, error) {
	pdf := newPDF()
	for i, card := range cards {
		if i%(cols*rows) == 0 {
			pdf.AddPage()
		}
		pos := i % (cols * rows)
		x := marginX + float64(pos%cols)*cardW
		y := float64(pos/cols) * cardH
		renderCard(pdf, card, x, y, cardW, cardH)
	}
	return output(pdf)
}

// GenerateSingleCard erzeugt ein einzelnes A7-Hochformat-PDF (74 mm × 105 mm).
func GenerateSingleCard(card CardData) ([]byte, error) {
	const sW, sH = 74.0, 105.0 // DIN A7 Hochformat
	pdf := fpdf.NewCustom(&fpdf.InitType{
		OrientationStr: "P",
		UnitStr:        "mm",
		Size:           fpdf.SizeType{Wd: sW, Ht: sH},
	})
	pdf.SetMargins(0, 0, 0)
	pdf.SetAutoPageBreak(false, 0)
	pdf.AddPage()
	renderCard(pdf, card, 0, 0, sW, sH)
	return output(pdf)
}

func newPDF() *fpdf.Fpdf {
	pdf := fpdf.New("L", "mm", "A4", "") // A4 Querformat für Sammel-PDF
	pdf.SetMargins(0, 0, 0)
	pdf.SetAutoPageBreak(false, 0)
	return pdf
}

func output(pdf *fpdf.Fpdf) ([]byte, error) {
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("pdf output: %w", err)
	}
	return buf.Bytes(), nil
}

var (
	reBold   = regexp.MustCompile(`\*\*(.+?)\*\*`)
	reItalic = regexp.MustCompile(`\*(.+?)\*`)
)

func stripMarkdown(s string) string {
	s = reBold.ReplaceAllString(s, "$1")
	s = reItalic.ReplaceAllString(s, "$1")
	s = strings.ReplaceAll(s, "- ", "\u2022 ")
	return s
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "\u2026"
}

// Zeichengrößen und Abstände
const (
	headerFontSize = 8 // pt
	labelFontSize  = 5 // pt
	descFontSize   = 4.5
	tableFontSize  = 5   // pt
	footerFontSize = 4   // pt
	tableLineH     = 2.1 // mm pro Textzeile (5 pt × 0,3528 mm/pt × ~1,2 Zeilenabstand)
	cellPadX       = 1.2 // mm horizontaler Innenabstand
	cellPadY       = 0.6 // mm vertikaler Innenabstand (oben)
)

// calcRowH berechnet die benötigte Zeilenhöhe mit korrektem Schriftstil pro Spalte.
// rightBold=true für die erste (hervorgehobene) Zeile einer Symptom-Tabelle.
func calcRowH(pdf *fpdf.Fpdf, leftText, rightText string, leftW, rightW float64, rightBold bool) float64 {
	pdf.SetFont(font, "B", tableFontSize) // linke Spalte: immer fett
	l := pdf.SplitLines([]byte(leftText), leftW-2*cellPadX)

	rStyle := ""
	if rightBold {
		rStyle = "B"
	}
	pdf.SetFont(font, rStyle, tableFontSize)
	r := pdf.SplitLines([]byte(rightText), rightW-2*cellPadX)

	n := len(l)
	if len(r) > n {
		n = len(r)
	}
	if n < 1 {
		n = 1
	}
	return float64(n)*tableLineH + 2*cellPadY
}

// renderCard zeichnet eine einzelne Karte in den Bereich (x,y) mit Breite cw und Höhe ch.
func renderCard(pdf *fpdf.Fpdf, card CardData, x, y, cw, ch float64) {
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	// Äußerer Rahmen
	pdf.SetLineWidth(0.3)
	pdf.SetDrawColor(100, 100, 100)
	pdf.Rect(x, y, cw, ch, "D")

	// Titelbalken
	titleH := 8.5
	if card.CardType == "symptom" {
		pdf.SetFillColor(30, 80, 140)
	} else {
		pdf.SetFillColor(20, 120, 80)
	}
	pdf.Rect(x, y, cw, titleH, "F")

	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont(font, "B", headerFontSize)
	pdf.SetXY(x+2, y+1)
	pdf.CellFormat(cw-4, titleH-2, tr(truncate(card.Title, 45)), "", 0, "LM", false, 0, "")

	pdf.SetFont(font, "", labelFontSize)
	label := "Leitsymptom"
	if card.CardType == "medication" {
		label = "Medikament"
	}
	pdf.SetXY(x+2, y+1)
	pdf.CellFormat(cw-4, titleH-2, tr(label), "", 0, "RM", false, 0, "")

	// Beschreibung unterhalb des Titels einfügen
	if card.Description != "" {
		descY := y + titleH - 3.0 // etwas unterhalb des Titels
		pdf.SetFont(font, "", descFontSize)
		pdf.SetTextColor(230, 230, 230)
		pdf.SetXY(x+2, descY)
		pdf.CellFormat(cw-4, 3, tr(truncate(card.Description, 120)), "", 0, "LM", false, 0, "")
		pdf.SetTextColor(255, 255, 255)
	}

	// Tabellenbereich
	footerH := 3.0
	tableTopY := y + titleH
	tableBottomY := y + ch - footerH

	leftColW := cw * 0.30 // linke Spalte schmaler (vorher 0.50)
	rightColW := cw - leftColW

	if card.CardType == "symptom" {
		renderSymptomTables(pdf, tr, card.Tables, x, tableTopY, tableBottomY, leftColW, rightColW)
	} else {
		renderEntries(pdf, tr, card.Entries, x, tableTopY, tableBottomY, leftColW, rightColW)
	}

	// Fußzeile
	footerY := y + ch - footerH
	pdf.SetFillColor(240, 240, 240)
	pdf.Rect(x, footerY, cw, footerH, "F")
	pdf.SetDrawColor(180, 180, 180)
	pdf.SetLineWidth(0.2)
	pdf.Line(x, footerY, x+cw, footerY)

	pdf.SetTextColor(110, 110, 110)
	pdf.SetFont(font, "", footerFontSize)

	src := ""
	if card.Source != "" {
		src = tr(truncate("Quelle: "+card.Source, 42))
	}
	pdf.SetXY(x+cellPadX, footerY+0.9)
	pdf.CellFormat(cw/2-cellPadX, footerH-1, src, "", 0, "LM", false, 0, "")

	pdf.SetXY(x+cw/2, footerY+0.9)
	pdf.CellFormat(cw/2-cellPadX, footerH-1, tr("Stand: "+card.UpdatedAt.Format("01/2006")), "", 0, "RM", false, 0, "")
}

// renderSymptomTables zeichnet mehrere benannte Tabellengruppen.
func renderSymptomTables(pdf *fpdf.Fpdf, tr func(string) string,
	tables []SymptomTableData, x, topY, bottomY, leftColW, rightColW float64) {

	if len(tables) == 0 {
		renderEmpty(pdf, tr, x, topY, leftColW+rightColW)
		return
	}

	tableW := leftColW + rightColW
	pdf.SetDrawColor(200, 200, 200)
	pdf.SetLineWidth(0.15)
	pdf.Line(x+leftColW, topY, x+leftColW, bottomY)

	curY := topY
	for ti, table := range tables {
		if curY >= bottomY {
			break
		}

		if ti > 0 {
			gapH := 2.5
			if curY+gapH > bottomY {
				break
			}
			// Graue Trennlinie mit leichtem Abstand
			pdf.SetDrawColor(160, 170, 185)
			pdf.SetLineWidth(0.4)
			pdf.Line(x+2, curY+gapH/2, x+tableW-2, curY+gapH/2)
			curY += gapH
		}

		if table.Title != "" {
			titleRowH := 3.8
			if curY+titleRowH > bottomY {
				break
			}
			pdf.SetFillColor(230, 235, 245)
			pdf.Rect(x, curY, tableW, titleRowH, "F")
			pdf.SetDrawColor(200, 205, 215)
			pdf.SetLineWidth(0.1)
			pdf.Line(x, curY, x+tableW, curY)

			pdf.SetTextColor(40, 60, 100)
			pdf.SetFont(font, "BI", tableFontSize)
			pdf.SetXY(x+cellPadX, curY+0.6)
			pdf.CellFormat(tableW-2*cellPadX, titleRowH-1, tr(truncate(table.Title, 50)), "", 0, "LM", false, 0, "")
			curY += titleRowH
		}

		for i, row := range table.Rows {
			leftText := tr(stripMarkdown(row.Medication))
			rightText := tr(stripMarkdown(row.RightCol))

			isFirst := i == 0
			rh := calcRowH(pdf, leftText, rightText, leftColW, rightColW, isFirst)

			if curY+rh > bottomY {
				break
			}

			if i > 0 || table.Title != "" {
				pdf.SetDrawColor(220, 220, 220)
				pdf.SetLineWidth(0.1)
				pdf.Line(x, curY, x+tableW, curY)
			}

			// Erste Zeile = Hauptmedikament → blaue Hinterlegung
			if isFirst {
				pdf.SetFillColor(205, 220, 245)
				pdf.Rect(x, curY, tableW, rh, "F")
			}

			// Linke Spalte – immer fett; erste Zeile: dunkelblau
			if isFirst {
				pdf.SetTextColor(20, 50, 110)
			} else {
				pdf.SetTextColor(40, 40, 40)
			}
			pdf.SetFont(font, "B", tableFontSize)
			pdf.SetXY(x+cellPadX, curY+cellPadY)
			pdf.MultiCell(leftColW-2*cellPadX, tableLineH, leftText, "", "LT", false)

			// Rechte Spalte – erste Zeile fett (blau), sonst regular
			if isFirst {
				pdf.SetTextColor(20, 50, 110)
				pdf.SetFont(font, "B", tableFontSize)
			} else {
				pdf.SetTextColor(40, 40, 40)
				pdf.SetFont(font, "", tableFontSize)
			}
			pdf.SetXY(x+leftColW+cellPadX, curY+cellPadY)
			pdf.MultiCell(rightColW-2*cellPadX, tableLineH, rightText, "", "LT", false)

			curY += rh
		}
	}
}

// renderEntries zeichnet eine einfache Key-Value-Tabelle (für Medikamente).
func renderEntries(pdf *fpdf.Fpdf, tr func(string) string,
	entries []models.CardEntry, x, topY, bottomY, leftColW, rightColW float64) {

	if len(entries) == 0 {
		renderEmpty(pdf, tr, x, topY, leftColW+rightColW)
		return
	}

	pdf.SetDrawColor(200, 200, 200)
	pdf.SetLineWidth(0.15)
	pdf.Line(x+leftColW, topY, x+leftColW, bottomY)

	curY := topY
	for i, entry := range entries {
		leftText := tr(stripMarkdown(entry.LeftCol))
		rightText := tr(stripMarkdown(entry.RightCol))

		rh := calcRowH(pdf, leftText, rightText, leftColW, rightColW, false)

		if curY+rh > bottomY {
			break
		}

		if i > 0 {
			pdf.SetDrawColor(220, 220, 220)
			pdf.SetLineWidth(0.1)
			pdf.Line(x, curY, x+leftColW+rightColW, curY)
		}

		pdf.SetTextColor(40, 40, 40)
		pdf.SetFont(font, "B", tableFontSize)
		pdf.SetXY(x+cellPadX, curY+cellPadY)
		pdf.MultiCell(leftColW-2*cellPadX, tableLineH, leftText, "", "LT", false)

		pdf.SetFont(font, "", tableFontSize)
		pdf.SetXY(x+leftColW+cellPadX, curY+cellPadY)
		pdf.MultiCell(rightColW-2*cellPadX, tableLineH, rightText, "", "LT", false)

		curY += rh
	}
}

func renderEmpty(pdf *fpdf.Fpdf, tr func(string) string, x, y, w float64) {
	pdf.SetTextColor(160, 160, 160)
	pdf.SetFont(font, "I", 7)
	pdf.SetXY(x+2, y+2)
	pdf.CellFormat(w-4, 6, tr("Keine Einträge"), "", 0, "LT", false, 0, "")
}
