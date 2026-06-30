package main

import (
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63")).Padding(0, 1)
	okBadge    = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	noBadge    = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
	subStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	warnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	helpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	paneStyle  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240"))
)

func tableStyles() table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.Bold(true).Foreground(lipgloss.Color("63"))
	s.Selected = s.Selected.Bold(true).Foreground(lipgloss.Color("231")).Background(lipgloss.Color("63"))
	return s
}

func (m Model) View() string {
	// Header: title + auth badge + optional spinner.
	badge := noBadge.Render("● not signed in")
	if m.authed {
		badge = okBadge.Render("● Entra: signed in")
	}
	header := lipgloss.JoinHorizontal(lipgloss.Center, titleStyle.Render("STRATUS"), "  ", badge)
	if m.busy != "" {
		header = lipgloss.JoinHorizontal(lipgloss.Center, header, "   ", m.spin.View()+" "+subStyle.Render(m.busy))
	}

	// Body.
	var body string
	switch m.view {
	case viewInstances:
		title := subStyle.Render("Instances — " + string(m.selected))
		body = lipgloss.JoinVertical(lipgloss.Left, title, paneStyle.Render(m.instTable.View()))
	default:
		body = paneStyle.Render(m.provTable.View())
	}

	// Status / warnings.
	lines := []string{header, "", body}
	if m.errMsg != "" {
		lines = append(lines, errStyle.Render("✗ "+m.errMsg))
	} else if m.msg != "" {
		lines = append(lines, subStyle.Render(m.msg))
	}
	if len(m.warnings) > 0 {
		lines = append(lines, warnStyle.Render("unavailable: "+strings.Join(m.warnings, " · ")))
	}
	lines = append(lines, "", helpStyle.Render(m.helpLine()))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m Model) helpLine() string {
	if m.view == viewInstances {
		return "↑/↓ navigate · esc back · r refresh · q quit"
	}
	return "↑/↓ navigate · l sign in · enter connect / open instances · r refresh · q quit"
}