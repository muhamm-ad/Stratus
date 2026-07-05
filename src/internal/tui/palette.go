package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textinput"
)

type command struct{ name, desc string }

var allCommands = []command{
	{"inventory", "go to inventory"}, {"sessions", "go to sessions"},
	{"audit", "go to audit"}, {"settings", "go to settings"},
	{"aws", "filter provider aws"}, {"azure", "filter provider azure"},
	{"gcp", "filter provider gcp"}, {"all", "clear provider filter"},
	{"running", "filter state running"}, {"stopped", "filter state stopped"},
	{"region", "region <name>"}, {"tag", "tag <k:v>"}, {"clear", "clear all filters"},
	{"connect", "connect to selection"}, {"stop", "stop selection"},
	{"refresh", "refresh inventory"}, {"reconnect", "reconnect <prov>"},
	{"theme", "theme <name>"}, {"logs", "toggle log pane"},
	{"help", "show help"}, {"quit", "sign out"},
}

type paletteModel struct {
	input    textinput.Model
	filtered []command
	cursor   int
}

func newPaletteModel() paletteModel {
	ti := textinput.New()
	ti.Prompt = ": "
	ti.Placeholder = "type a command…"
	return paletteModel{input: ti, filtered: allCommands}
}

func (m *paletteModel) open() tea.Cmd {
	m.input.SetValue("")
	m.filtered = allCommands
	m.cursor = 0
	return m.input.Focus()
}

func (m paletteModel) Update(msg tea.KeyPressMsg) (paletteModel, *command) {
	switch msg.String() {
	case "up", "ctrl+k":
		if m.cursor > 0 { m.cursor-- }
	case "down", "ctrl+j":
		if m.cursor < len(m.filtered)-1 { m.cursor++ }
	case "enter":
		if len(m.filtered) > 0 { c := m.filtered[m.cursor]; return m, &c }
	default:
		m.input, _ = m.input.Update(msg)
		q := strings.ToLower(m.input.Value())
		m.filtered = m.filtered[:0]
		for _, c := range allCommands {
			if strings.Contains(c.name, q) || strings.Contains(c.desc, q) {
				m.filtered = append(m.filtered, c)
			}
		}
		m.cursor = 0
	}
	return m, nil
}
