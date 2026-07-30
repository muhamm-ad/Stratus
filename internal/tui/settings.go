package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/muhamm-ad/stratus/internal/core"
	"github.com/muhamm-ad/stratus/internal/service"
)

type settingsModel struct {
	svc         *service.Service
	styles      Styles
	cursor      int
	autoRefresh bool
}

func newSettingsModel(svc *service.Service, s Styles) settingsModel {
	return settingsModel{svc: svc, styles: s, autoRefresh: true}
}

type settingsCmd struct{ themeIdx int }

func (m *settingsModel) applyStyles(s Styles) {
	m.styles = s
}

// maxCursor is the last valid row index: one row per provider, then
// auto-refresh, then theme.
func (m *settingsModel) maxCursor() int {
	return len(m.svc.GetCloudProvidersIDs()) + 1
}

func (m *settingsModel) Update(msg tea.KeyPressMsg, themeIdx int, s Styles) (settingsModel, *settingsCmd, appIntent) {
	m.applyStyles(s)
	switch msg.String() {
	case "j", "down":
		if m.cursor < m.maxCursor() {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "enter":
		providerIDs := m.svc.GetCloudProvidersIDs()
		switch {
		case m.cursor < len(providerIDs):
			return *m, nil, appIntent{kind: intentReconnect, provider: string(providerIDs[m.cursor])}
		case m.cursor == len(providerIDs):
			m.autoRefresh = !m.autoRefresh
		case m.cursor == len(providerIDs)+1:
			idx := (themeIdx + 1) % len(Themes)
			return *m, &settingsCmd{themeIdx: idx}, appIntent{}
		}
	}
	return *m, nil, appIntent{}
}

func (m *settingsModel) View(w, h int, themeIdx int) string {
	head := m.styles.SectionHead.Render("PROVIDERS & PREFERENCES · j/k move · ⏎ toggle/cycle/reconnect")
	return m.renderSettings(w, head, themeIdx)
}

func (m *settingsModel) renderSettings(w int, head string, themeIdx int) string {
	var rows []string
	providerIDs := m.svc.GetCloudProvidersIDs()
	for i, id := range providerIDs {
		val, valStyle, hint := providerRowInfo(m.styles, m.svc.GetCloudProviderStatus(id))
		rows = append(rows, settingsRow(m.styles, m.cursor == i, string(id), val, valStyle, hint))
	}

	refresh := "[off]"
	if m.autoRefresh {
		refresh = "[on] every 60s"
	}
	rows = append(rows,
		settingsRow(m.styles, m.cursor == len(providerIDs), "auto-refresh", refresh, m.styles.Dim, "⏎ toggle"),
		settingsRow(m.styles, m.cursor == len(providerIDs)+1, "theme", Themes[themeIdx].Name+" (charm · stratus · mono · terminal)", m.styles.Dim, "⏎ cycle"),
	)

	body := head + "\n\n" + strings.Join(rows, "\n")
	return lipgloss.NewStyle().Width(w).Render(body)
}

// providerRowInfo maps a provider's live connection state to the row's
// value text/style/hint. The mockup's fabricated "· {identity}" suffix is
// dropped — Stratus has no per-provider identity-label concept today.
func providerRowInfo(s Styles, st core.CloudProviderStatus) (val string, valStyle lipgloss.Style, hint string) {
	switch st {
	case core.CloudProviderStatusError:
		return "session expired", s.Err, "R reconnect"
	case core.CloudProviderStatusAuthenticating:
		return SpinnerFrames[0] + " authorizing…", s.Warn, ""
	case core.CloudProviderStatusAuthenticated:
		return "connected", s.OK, "⏎ reconnect"
	default:
		return "not yet synced", s.Dim, "⏎ reconnect"
	}
}

func settingsRow(s Styles, selected bool, label, val string, valStyle lipgloss.Style, hint string) string {
	cur := "  "
	if selected {
		cur = s.Cursor.Render("▸ ")
	}
	labelCol := s.Text.Bold(true).Width(14).Render(label)
	line := cur + labelCol + " " + valStyle.Render(val)
	if hint != "" {
		line += "  " + s.Dim.Render(hint)
	}
	return line
}