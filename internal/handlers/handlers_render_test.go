package handlers

import (
	"strings"
	"testing"
)

func TestRenderMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		contains string
	}{
		{"bold", "**bold**", "<strong>bold</strong>"},
		{"line break", "line1\nline2", "<br"},
		{"list", "- item1\n- item2", "<li>"},
		{"empty", "", ""},
		{"plain", "plain text", "plain text"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(renderMarkdown(tt.in))
			if tt.contains != "" && !strings.Contains(got, tt.contains) {
				t.Errorf("renderMarkdown(%q) = %q, should contain %q", tt.in, got, tt.contains)
			}
		})
	}
}
