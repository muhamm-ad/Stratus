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
	logAlerts   bool
}

func newSettingsModel(svc *service.Service, s Styles) settingsModel {
	return settingsModel{svc: svc, styles: s, autoRefresh: true, logAlerts: true}
}

type settingsCmd struct{ themeIdx int }

func (m *settingsModel) applyStyles(s Styles) {
	m.styles = s
}

// maxCursor is the last valid row index: one row per provider, then
// auto-refresh, log alerts, then theme.
func (m *settingsModel) maxCursor() int {
	return len(m.svc.GetCloudProvidersIDs()) + 2
}

func (m *settingsModel) Update(msg tea.KeyPressMsg, themeIdx int, s Styles, k KeyMap) (settingsModel, *settingsCmd, appIntent) {
	m.applyStyles(s)
	switch k.Nav.Match(msg) {
	case ActionMoveDown:
		if m.cursor < m.maxCursor() {
			m.cursor++
		}
	case ActionMoveUp:
		if m.cursor > 0 {
			m.cursor--
		}
	case ActionJumpTop:
		m.cursor = 0
	case ActionJumpBottom:
		m.cursor = m.maxCursor()
	case ActionSelect:
		providerIDs := m.svc.GetCloudProvidersIDs()
		n := len(providerIDs)
		switch {
		case m.cursor < n:
			return *m, nil, appIntent{kind: intentReconnect, provider: string(providerIDs[m.cursor])}
		case m.cursor == n:
			m.autoRefresh = !m.autoRefresh
		case m.cursor == n+1:
			m.logAlerts = !m.logAlerts
		case m.cursor == n+2:
			idx := (themeIdx + 1) % len(Themes)
			return *m, &settingsCmd{themeIdx: idx}, appIntent{}
		}
	}
	return *m, nil, appIntent{}
}

func (m settingsModel) sidebarBody(s Styles, innerW int, themeIdx int) string {
	var rows []string
	providerIDs := m.svc.GetCloudProvidersIDs()
	for i, id := range providerIDs {
		val, valStyle, hint := providerRowInfo(s, m.svc.GetCloudProviderStatus(id))
		rows = append(rows, settingsRow(s, m.cursor == i, string(id), val, valStyle, hint))
	}

	refresh := "[off]"
	if m.autoRefresh {
		refresh = "[on] every 60s"
	}
	alerts := "[off]"
	if m.logAlerts {
		alerts = "[on] toast each log"
	}
	n := len(providerIDs)
	rows = append(rows,
		settingsRow(s, m.cursor == n, "auto-refresh", refresh, s.Dim, "⏎ toggle"),
		settingsRow(s, m.cursor == n+1, "log alerts", alerts, s.Dim, "⏎ toggle"),
		settingsRow(s, m.cursor == n+2, "theme", Themes[themeIdx].Name, s.Dim, "⏎ cycle"),
	)

	out := make([]string, len(rows))
	for i, row := range rows {
		out[i] = clipLine(row, innerW)
	}
	return strings.Join(out, "\n")
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
