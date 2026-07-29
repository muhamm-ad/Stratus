package tui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type command struct{ name, desc string }

var allCommands = []command{
	{"inventory", "go to inventory"},
	{"sessions", "go to sessions"},
	{"audit", "go to audit"},
	{"settings", "go to settings"},
	{"aws", "filter provider aws"},
	{"azure", "filter provider azure"},
	{"gcp", "filter provider gcp"},
	{"all", "clear provider filter"},
	{"running", "filter state running"},
	{"stopped", "filter state stopped"},
	{"region", "region <name>"},
	{"tag", "tag <k:v>"},
	{"clear", "clear all filters"},
	{"connect", "connect to selection"},
	{"start", "start selection"},
	{"stop", "stop selection"},
	{"refresh", "refresh inventory"},
	{"reconnect", "reconnect <prov>"},
	{"theme", "theme <name>"},
	{"logs", "toggle log pane"},
	{"help", "show help"},
	{"quit", "sign out"},
}

type paletteItem command

func (c paletteItem) FilterValue() string { return c.name + " " + c.desc }

func commandItems(cmds []command) []list.Item {
	items := make([]list.Item, len(cmds))
	for i, c := range cmds {
		items[i] = paletteItem(c)
	}
	return items
}

// paletteListKeyMap only claims up/down — bare letters stay typeable in the filter.
func paletteListKeyMap() list.KeyMap {
	return list.KeyMap{
		CursorUp:   key.NewBinding(key.WithKeys("up", "ctrl+k")),
		CursorDown: key.NewBinding(key.WithKeys("down", "ctrl+j")),
	}
}

type paletteModel struct {
	input  textinput.Model
	styles Styles
	list   list.Model
}

func newPaletteModel(s Styles) paletteModel {
	ti := textinput.New()
	ti.Prompt = ": "
	ti.Placeholder = "type a command…"

	l := list.New(commandItems(allCommands), list.NewDefaultDelegate(), 48, len(allCommands))
	l.KeyMap = paletteListKeyMap()
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.SetFilteringEnabled(false)
	l.DisableQuitKeybindings()
	return paletteModel{input: ti, styles: s, list: l}
}

func (m *paletteModel) applyStyles(s Styles) { m.styles = s }

func (m *paletteModel) open() tea.Cmd {
	m.input.SetValue("")
	m.list.SetItems(commandItems(allCommands))
	m.list.Select(0)
	return m.input.Focus()
}

func (m paletteModel) Update(msg tea.KeyPressMsg) (paletteModel, *command) {
	switch msg.String() {
	case "up", "ctrl+k", "down", "ctrl+j":
		m.list, _ = m.list.Update(msg)
	case "enter":
		if item, ok := m.list.SelectedItem().(paletteItem); ok {
			c := command(item)
			return m, &c
		}
	default:
		m.input, _ = m.input.Update(msg)
		q := strings.ToLower(m.input.Value())
		var filtered []command
		for _, c := range allCommands {
			if strings.Contains(c.name, q) || strings.Contains(c.desc, q) {
				filtered = append(filtered, c)
			}
		}
		m.list.SetItems(commandItems(filtered))
		m.list.Select(0)
	}
	return m, nil
}

func (m paletteModel) View() string {
	s := m.styles
	var rows []string
	rows = append(rows, s.Accent.Bold(true).Render("command palette"), "", s.Dim.Render(m.input.View()), "")
	if len(m.list.Items()) == 0 {
		rows = append(rows, s.Dim.Render("no matches"))
	} else {
		for i, item := range m.list.Items() {
			c := item.(paletteItem)
			cur := "  "
			if i == m.list.Index() {
				cur = s.Cursor.Render("▸ ")
			}
			rows = append(rows, cur+s.Text.Render(c.name)+s.Dim.Render(" — "+c.desc))
		}
	}
	rows = append(rows, "", s.Dim.Render("⏎ run · esc cancel"))
	return s.OverlayBox.Width(52).Render(strings.Join(rows, "\n"))
}
