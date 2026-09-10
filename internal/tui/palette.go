package tui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	{"logs", "toggle log alerts"},
	{"sidebar", "toggle sidebar"},
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

const palettePreferredWidth = 52

type paletteModel struct {
	input  textinput.Model
	styles Styles
	list   list.Model
	offset int
	visH   int
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
	m.offset = 0
	return m.input.Focus()
}

func (m *paletteModel) moveBy(delta int) {
	n := len(m.list.Items())
	if n == 0 {
		return
	}
	idx := m.list.Index() + delta
	if idx < 0 {
		idx = 0
	}
	if idx >= n {
		idx = n - 1
	}
	m.list.Select(idx)
}

func (m paletteModel) Update(msg tea.KeyPressMsg) (paletteModel, *command) {
	switch msg.String() {
	case "up", "ctrl+k", "down", "ctrl+j":
		m.list, _ = m.list.Update(msg)
	case "pgup":
		m.moveBy(-max(1, m.visH))
	case "pgdown":
		m.moveBy(max(1, m.visH))
	case "home":
		if n := len(m.list.Items()); n > 0 {
			m.list.Select(0)
		}
	case "end":
		if n := len(m.list.Items()); n > 0 {
			m.list.Select(n - 1)
		}
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
		m.offset = 0
	}
	return m, nil
}

func (m *paletteModel) clampOffset(visible, total int) {
	if total <= visible {
		m.offset = 0
		return
	}
	idx := m.list.Index()
	if idx < m.offset {
		m.offset = idx
	}
	if idx >= m.offset+visible {
		m.offset = idx - visible + 1
	}
	m.offset = max(0, min(m.offset, total-visible))
}

func (m *paletteModel) View(w, h int) string {
	s := m.styles
	maxW := max(24, w-2)
	maxH := max(8, h-2)
	boxW := min(palettePreferredWidth, maxW)

	frameH := s.Dialog.GetHorizontalFrameSize()
	frameV := s.Dialog.GetVerticalFrameSize()
	innerW := max(8, boxW-frameH)
	// title, blank, input, blank, list, blank, footer
	chrome := 6
	maxListH := max(1, maxH-frameV-chrome)

	items := m.list.Items()
	n := len(items)
	listH := maxListH
	if n == 0 {
		listH = 1
	} else if n < maxListH {
		listH = n
	}
	scrollable := n > listH
	m.visH = listH
	m.clampOffset(listH, n)

	itemW := innerW
	if scrollable {
		itemW = max(8, innerW-scrollbarCols)
	}

	var listBody string
	if n == 0 {
		listBody = s.DialogKey.Render("no matches")
	} else {
		end := min(m.offset+listH, n)
		rows := make([]string, 0, listH)
		for i := m.offset; i < end; i++ {
			c := items[i].(paletteItem)
			cur := "  "
			if i == m.list.Index() {
				cur = s.Cursor.Render("▸ ")
			}
			row := cur + s.Text.Render(c.name) + s.Dim.Render(" — "+c.desc)
			rows = append(rows, clipLine(row, itemW))
		}
		listBody = strings.Join(rows, "\n")
		if scrollable {
			listBody = joinScrollbar(listBody, innerW, listH, n, m.offset, s)
		}
	}

	footer := "⏎ run · esc cancel"
	if scrollable {
		footer = "↑↓ scroll · ⏎ run · esc cancel"
	}

	inner := lipgloss.JoinVertical(lipgloss.Left,
		s.DialogTitle.Render("Command palette"),
		"",
		s.DialogBody.Render(clipLine(m.input.View(), innerW)),
		"",
		listBody,
		"",
		s.DialogKey.Render(clipLine(footer, innerW)),
	)
	return boxNoWrap(s.Dialog, inner, boxW, min(maxH, max(1, lipgloss.Height(inner)+s.Dialog.GetVerticalFrameSize())))
}
