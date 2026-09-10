package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type helpEntry struct {
	keys string
	desc string
}

type helpCategory struct {
	title   string
	entries []helpEntry
}

type helpModel struct {
	scroll int
}

func newHelpModel() helpModel { return helpModel{} }

func helpEntryFrom(b Binding, desc string) helpEntry {
	return helpEntry{keys: b.Keys.Help().Key, desc: desc}
}

func helpCategories(k KeyMap) []helpCategory {
	return []helpCategory{
		{
			title: "General",
			entries: []helpEntry{
				helpEntryFrom(k.Global.Help, "show / hide this help"),
				helpEntryFrom(k.Global.Palette, "open the command palette"),
				helpEntryFrom(k.Global.Theme, "cycle color theme"),
				helpEntryFrom(k.Global.Logs, "toggle log toasts"),
				helpEntryFrom(k.Global.Quit, "quit (close app properly)"),
			},
		},
		{
			title: "Tabs",
			entries: []helpEntry{
				helpEntryFrom(k.Global.TabInventory, "inventory"),
				helpEntryFrom(k.Global.TabSessions, "sessions"),
				helpEntryFrom(k.Global.TabAudit, "audit"),
				helpEntryFrom(k.Global.TabSettings, "settings"),
			},
		},
		{
			title: "Navigation",
			entries: []helpEntry{
				helpEntryFrom(k.Nav.Up, "move up"),
				helpEntryFrom(k.Nav.Down, "move down"),
				helpEntryFrom(k.Nav.Enter, "confirm / select"),
				helpEntryFrom(k.Nav.Esc, "close / dismiss"),
				helpEntryFrom(k.Overlay.Next, "switch to next button"),
				helpEntryFrom(k.Overlay.Prev, "switch to previous button"),
			},
		},
		{
			title: "Inventory",
			entries: []helpEntry{
				helpEntryFrom(k.Inventory.Mark, "mark this vm"),
				helpEntryFrom(k.Inventory.MarkAll, "mark all / none"),
				helpEntryFrom(k.Inventory.Connect, "connect to selection"),
				helpEntryFrom(k.Inventory.Start, "start selection"),
				helpEntryFrom(k.Inventory.Stop, "stop selection"),
				helpEntryFrom(k.Inventory.Search, "filter by name"),
				helpEntryFrom(k.Inventory.SortKey, "cycle sort column"),
				helpEntryFrom(k.Inventory.SortDir, "reverse sort"),
				helpEntryFrom(k.Inventory.FilterProv, "cycle provider"),
				helpEntryFrom(k.Inventory.FilterState, "cycle state"),
				helpEntryFrom(k.Inventory.FilterRegion, "cycle region"),
				helpEntryFrom(k.Inventory.ClearFilters, "clear filters"),
				helpEntryFrom(k.Inventory.Refresh, "refresh inventory"),
				helpEntryFrom(k.Global.Reconnect, "reconnect expired provider"),
			},
		},
	}
}

func (m helpModel) View(s Styles, k KeyMap, w, h int) string {
	stacked := helpShouldStack(s, k, w)
	body := helpBody(s, k, stacked)
	bodyLines := strings.Split(body, "\n")
	bodyH := helpBodyHeight(s, h)
	scrollable := len(bodyLines) > bodyH
	scroll := 0

	footer := keyed(k.Nav.Esc, "close")
	if scrollable {
		footer = k.Nav.Up.Keys.Help().Key + " " + keyed(k.Nav.Down, "scroll") + " · " + keyed(k.Nav.Esc, "close")
	}

	title := s.DialogTitle.Render("Help")
	footerView := s.DialogKey.Render(strings.TrimSpace(footer))
	maxW := max(20, w-2)
	maxH := max(8, h-2)
	if scrollable {
		maxScroll := max(0, len(bodyLines)-bodyH)
		scroll = max(0, min(m.scroll, maxScroll))
		end := min(scroll+bodyH, len(bodyLines))
		innerMax := max(1, maxW-s.Dialog.GetHorizontalFrameSize())
		frameW := min(innerMax, max(
			lipgloss.Width(title),
			lipgloss.Width(footerView),
			lipgloss.Width(body)+scrollbarCols,
		))
		body = joinScrollbar(strings.Join(bodyLines[scroll:end], "\n"), frameW, bodyH, len(bodyLines), scroll, s)
	}

	return s.Dialog.Align(lipgloss.Left).MaxWidth(maxW).MaxHeight(maxH).Render(lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		body,
		"",
		footerView,
	))
}

func helpShouldStack(s Styles, k KeyMap, w int) bool {
	twoCol := helpBody(s, k, false)
	return lipgloss.Width(twoCol)+scrollbarCols+s.Dialog.GetHorizontalFrameSize() > w
}

func helpBody(s Styles, k KeyMap, stacked bool) string {
	cats := helpCategories(k)
	if stacked {
		return renderHelpColumn(s, cats)
	}
	mid := 3 //len(cats) / 2
	left := renderHelpColumn(s, cats[:mid])
	right := renderHelpColumn(s, cats[mid:])
	return lipgloss.JoinHorizontal(lipgloss.Top, left, "    ", right)
}

func helpBodyHeight(s Styles, h int) int {
	chrome := s.Dialog.GetVerticalFrameSize() +
		lipgloss.Height(s.DialogTitle.Render("Help")) +
		lipgloss.Height(s.DialogKey.Render("esc close")) +
		2 // blank lines around the body
	return max(3, h-chrome-2)
}

func (m *helpModel) maxScroll(s Styles, k KeyMap, w, h int) int {
	lines := lipgloss.Height(helpBody(s, k, helpShouldStack(s, k, w)))
	return max(0, lines-helpBodyHeight(s, h))
}

func (m *helpModel) scrollBy(delta int, s Styles, k KeyMap, w, h int) {
	m.scroll += delta
	maxScroll := m.maxScroll(s, k, w, h)
	if m.scroll < 0 {
		m.scroll = 0
	}
	if m.scroll > maxScroll {
		m.scroll = maxScroll
	}
}

func renderHelpColumn(s Styles, cats []helpCategory) string {
	keyW := 0
	for _, cat := range cats {
		for _, e := range cat.entries {
			keyW = max(keyW, lipgloss.Width(e.keys))
		}
	}
	keyStyle := s.HelpKey.Width(keyW)

	rows := make([]string, 0, 32)
	for i, cat := range cats {
		if i > 0 {
			rows = append(rows, "")
		}
		rows = append(rows, s.HelpCategory.Render(cat.title))
		for _, e := range cat.entries {
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
				keyStyle.Render(e.keys),
				"  ",
				s.HelpDesc.Render(e.desc),
			))
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}
