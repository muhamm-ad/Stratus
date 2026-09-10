package tui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestOpaqueOverlayPadsToRectangle(t *testing.T) {
	out := opaqueOverlay("ab\n", lipgloss.Color("#123456"))
	if got, want := lipgloss.Width(out), 2; got != want {
		t.Fatalf("width = %d, want %d", got, want)
	}
	if got, want := lipgloss.Height(out), 2; got != want {
		t.Fatalf("height = %d, want %d", got, want)
	}
	for i, line := range strings.Split(ansi.Strip(out), "\n") {
		if lipgloss.Width(line) != 2 {
			t.Fatalf("line %d width = %d, want 2 (%q)", i, lipgloss.Width(line), line)
		}
	}
}
