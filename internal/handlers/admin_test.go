package handlers

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestParseEntries(t *testing.T) {
	form := url.Values{
		"entry_left[]":  {"Wirkstoff", "Dosierung", "", "Hinweis"},
		"entry_right[]": {"Epinephrin", "0,5 mg", "", "Cave Allergie"},
	}
	r := &http.Request{Form: form}

	entries := parseEntries(r)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries (empty skipped), got %d", len(entries))
	}
	if entries[0].LeftCol != "Wirkstoff" || entries[0].RightCol != "Epinephrin" {
		t.Errorf("entry 0 = %+v", entries[0])
	}
	if entries[2].LeftCol != "Hinweis" {
		t.Errorf("entry 2 = %+v", entries[2])
	}
}

func TestParseEntriesUnequalLengths(t *testing.T) {
	form := url.Values{
		"entry_left[]":  {"Key1", "Key2"},
		"entry_right[]": {"Val1"},
	}
	r := &http.Request{Form: form}

	entries := parseEntries(r)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[1].RightCol != "" {
		t.Errorf("second entry right should be empty, got %q", entries[1].RightCol)
	}
}

func TestParseEntriesEmpty(t *testing.T) {
	r := &http.Request{Form: url.Values{}}
	entries := parseEntries(r)
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestParseEntriesTrimWhitespace(t *testing.T) {
	form := url.Values{
		"entry_left[]":  {"  Wirkstoff  "},
		"entry_right[]": {"  Epinephrin  "},
	}
	r := &http.Request{Form: form}

	entries := parseEntries(r)
	if entries[0].LeftCol != "Wirkstoff" || entries[0].RightCol != "Epinephrin" {
		t.Errorf("expected trimmed values, got %+v", entries[0])
	}
}

func TestParseSymptomTables(t *testing.T) {
	form := url.Values{
		"table_count":   {"2"},
		"table_0_title": {"Erstlinientherapie"},
		"row_count_0":   {"2"},
		"row_0_0_med":   {"Adrenalin"},
		"row_0_0_right": {"0,5 mg i.m."},
		"row_0_1_med":   {""},
		"row_0_1_right": {""},
		"table_1_title": {""},
		"row_count_1":   {"1"},
		"row_1_0_med":   {"Volumen"},
		"row_1_0_right": {"500 ml"},
	}

	body := strings.NewReader(form.Encode())
	r, _ := http.NewRequest(http.MethodPost, "/", body)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ParseForm()

	tables := parseSymptomTables(r)
	if len(tables) != 2 {
		t.Fatalf("expected 2 tables, got %d", len(tables))
	}
	if tables[0].Title != "Erstlinientherapie" {
		t.Errorf("table 0 title = %q", tables[0].Title)
	}
	if len(tables[0].Rows) != 1 {
		t.Errorf("table 0 should have 1 row (empty skipped), got %d", len(tables[0].Rows))
	}
	if tables[1].Title != "" {
		t.Errorf("table 1 should have empty title, got %q", tables[1].Title)
	}
}

func TestParseSymptomTablesEmpty(t *testing.T) {
	form := url.Values{"table_count": {"0"}}
	body := strings.NewReader(form.Encode())
	r, _ := http.NewRequest(http.MethodPost, "/", body)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ParseForm()

	tables := parseSymptomTables(r)
	if len(tables) != 0 {
		t.Errorf("expected 0 tables, got %d", len(tables))
	}
}

func TestParseSymptomTablesSkipsEmptyTable(t *testing.T) {
	form := url.Values{
		"table_count":   {"1"},
		"table_0_title": {""},
		"row_count_0":   {"1"},
		"row_0_0_med":   {""},
		"row_0_0_right": {""},
	}
	body := strings.NewReader(form.Encode())
	r, _ := http.NewRequest(http.MethodPost, "/", body)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ParseForm()

	tables := parseSymptomTables(r)
	if len(tables) != 0 {
		t.Errorf("expected 0 tables (all empty), got %d", len(tables))
	}
}

func TestParseMedicationIDs(t *testing.T) {
	form := url.Values{
		"medication_ids[]": {"1", "5", "invalid", "42"},
	}
	r := &http.Request{Form: form}

	ids := parseMedicationIDs(r)
	if len(ids) != 3 {
		t.Fatalf("expected 3 valid IDs, got %d", len(ids))
	}
	if ids[0] != 1 || ids[1] != 5 || ids[2] != 42 {
		t.Errorf("ids = %v", ids)
	}
}

func TestParseMedicationIDsEmpty(t *testing.T) {
	r := &http.Request{Form: url.Values{}}
	ids := parseMedicationIDs(r)
	if len(ids) != 0 {
		t.Errorf("expected 0 IDs, got %d", len(ids))
	}
}

func TestParseSymptomIDs(t *testing.T) {
	form := url.Values{
		"symptom_ids[]": {"2", "7", "bad", "99"},
	}
	r := &http.Request{Form: form}

	ids := parseSymptomIDs(r)
	if len(ids) != 3 {
		t.Fatalf("expected 3 valid IDs, got %d", len(ids))
	}
	if ids[0] != 2 || ids[1] != 7 || ids[2] != 99 {
		t.Errorf("ids = %v", ids)
	}
}

func TestParseSymptomIDsEmpty(t *testing.T) {
	r := &http.Request{Form: url.Values{}}
	ids := parseSymptomIDs(r)
	if len(ids) != 0 {
		t.Errorf("expected 0 IDs, got %d", len(ids))
	}
}
