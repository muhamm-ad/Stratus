package tui

import "charm.land/lipgloss/v2"

type helpModel struct{}

func newHelpModel() helpModel { return helpModel{} }

func (helpModel) View(s Styles) string {
	sections := []struct{ title, body string }{
		{"NAVIGATION", "1 inventory · 2 sessions · 3 audit · 4 settings · q quit"},
		{"INVENTORY", HintInventory},
		{"SESSIONS", HintSessions},
		{"AUDIT", HintAudit},
		{"SETTINGS", HintSettings},
		{"GLOBAL", "/ filter · : command palette · L logs · t theme · ? help"},
	}
	var rows []string
	rows = append(rows, s.Accent.Bold(true).Render("keyboard shortcuts"))
	rows = append(rows, "")
	for _, sec := range sections {
		rows = append(rows, s.SectionHead.Render(sec.title))
		rows = append(rows, s.Dim.Render(sec.body))
		rows = append(rows, "")
	}
	rows = append(rows, s.Dim.Render("esc close"))
	return s.OverlayBox.Width(64).Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}
