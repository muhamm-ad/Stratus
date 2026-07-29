package tui

import (
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/muhamm-ad/stratus/internal/service"
)

type CLIStatus struct {
	Name     string // aws-cli, session-manager-plugin, az-cli, gcloud
	Bin      string // for exec.LookPath
	Detected bool
	Hint     string // "install to connect"
}

// settingsModel renders providers, auto-refresh, theme, and LOCAL CLI DETECTION.
type settingsModel struct {
	svc         *service.Service
	styles      Styles
	cursor      int
	autoRefresh bool
	clis        []CLIStatus
}

func newSettingsModel(svc *service.Service, s Styles) settingsModel {
	clis := detectCLIs()
	return settingsModel{svc: svc, styles: s, autoRefresh: true, clis: clis}
}

type settingsCmd struct{ themeIdx int }

func (m *settingsModel) applyStyles(s Styles) {
	m.styles = s
}

func (m *settingsModel) Update(msg tea.KeyPressMsg, themeIdx int, s Styles) (settingsModel, *settingsCmd) {
	m.applyStyles(s)
	switch msg.String() {
	case "j", "down":
		if m.cursor < 3 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "enter":
		switch m.cursor {
		case 1:
			m.autoRefresh = !m.autoRefresh
		case 2:
			idx := (themeIdx + 1) % len(Themes)
			return *m, &settingsCmd{themeIdx: idx}
		}
	}
	return *m, nil
}

func (m *settingsModel) View(w, h int, themeIdx int) string {
	head := m.styles.SectionHead.Render("PROVIDERS & PREFERENCES · j/k move · ⏎ toggle/cycle")
	cliHead := m.styles.SectionHead.Render("LOCAL CLI DETECTION")
	var cliRows []string
	for _, c := range m.clis {
		if c.Detected {
			cliRows = append(cliRows, m.styles.OK.Render("✓ ")+m.styles.Text.Render(c.Name+" detected"))
		} else {
			cliRows = append(cliRows, m.styles.Err.Render("✗ ")+m.styles.Text.Render(c.Name+" missing — "+c.Hint))
		}
	}
	return m.renderSettings(w, head, cliHead, cliRows, themeIdx)
}

func (m *settingsModel) renderSettings(w int, head, cliHead string, cliRows []string, themeIdx int) string {
	refresh := "[off]"
	if m.autoRefresh {
		refresh = "[on] every 60s"
	}
	rows := []string{
		row(m.styles, m.cursor == 0, "providers", "aws · azure · gcp — status via inventory sync"),
		row(m.styles, m.cursor == 1, "auto-refresh", refresh),
		row(m.styles, m.cursor == 2, "theme", Themes[themeIdx].Name+" (charm · stratus · mono · terminal)"),
	}
	body := head + "\n\n" + strings.Join(rows, "\n") + "\n\n" + cliHead + "\n" + strings.Join(cliRows, "\n")
	return lipgloss.NewStyle().Width(w).Render(body)
}

func row(s Styles, selected bool, label, value string) string {
	cur := "  "
	if selected {
		cur = s.Cursor.Render("▸ ")
	}
	return cur + s.Text.Render(label+": ") + s.Dim.Render(value)
}

func detectCLIs() []CLIStatus {
	det := func(name, bin, hint string) CLIStatus {
		_, err := exec.LookPath(bin)
		return CLIStatus{Name: name, Bin: bin, Detected: err == nil, Hint: hint}
	}
	return []CLIStatus{
		det("aws-cli", "aws", "install to connect"),
		det("session-manager-plugin", "session-manager-plugin", "install to connect"),
		det("az-cli", "az", "install to connect"),
		det("gcloud", "gcloud", "install to connect"),
	}
}
