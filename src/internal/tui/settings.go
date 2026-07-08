package tui

import (
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/muhamm-ad/stratus/service"
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
	cursor      int
	autoRefresh bool
	clis        []CLIStatus
}

func newSettingsModel(svc *service.Service) settingsModel {
	clis := detectCLIs()
	return settingsModel{svc: svc, autoRefresh: true, clis: clis}
}

type settingsCmd struct{ themeIdx int }

func (m settingsModel) Update(msg tea.KeyPressMsg, themeIdx int) (settingsModel, *settingsCmd) {
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
			return m, &settingsCmd{themeIdx: idx}
		}
	}
	return m, nil
}

func (m settingsModel) View(s Styles, w, h int, themeIdx int) string {
	head := s.SectionHead.Render("PROVIDERS & PREFERENCES · j/k move · ⏎ toggle/cycle")
	cliHead := s.SectionHead.Render("LOCAL CLI DETECTION")
	var cliRows []string
	for _, c := range m.clis {
		if c.Detected {
			cliRows = append(cliRows, s.OK.Render("✓ ")+s.Text.Render(c.Name+" detected"))
		} else {
			cliRows = append(cliRows, s.Err.Render("✗ ")+s.Text.Render(c.Name+" missing — "+c.Hint))
		}
	}
	return renderSettings(s, w, head, cliHead, cliRows, m, themeIdx)
}

func renderSettings(s Styles, w int, head, cliHead string, cliRows []string, m settingsModel, themeIdx int) string {
	refresh := "[off]"
	if m.autoRefresh {
		refresh = "[on] every 60s"
	}
	rows := []string{
		row(s, m.cursor == 0, "providers", "aws · azure · gcp — status via inventory sync"),
		row(s, m.cursor == 1, "auto-refresh", refresh),
		row(s, m.cursor == 2, "theme", Themes[themeIdx].Name+" (charm · stratus · mono)"),
	}
	body := head + "\n\n" + strings.Join(rows, "\n") + "\n\n" + cliHead + "\n" + strings.Join(cliRows, "\n")
	return lipgloss.NewStyle().Width(w).Render(body)
}

func row(s Styles, selected bool, label, value string) string {
	cur := "  "
	if selected {
		cur = s.Accent.Render("▸ ")
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
