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
	query  bool // filter/query panel instead of the app help
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
				helpEntryFrom(k.Inventory.QueryHelp, "query / filter help"),
				helpEntryFrom(k.Inventory.Refresh, "refresh inventory"),
				helpEntryFrom(k.Global.Reconnect, "reconnect expired provider"),
			},
		},
		{
			title: "Sidebar",
			entries: []helpEntry{
				helpEntryFrom(k.Global.Sidebar, "show / hide the sidebar"),
				helpEntryFrom(k.Sidebar.TabNotif, "notifications tab (while open)"),
				helpEntryFrom(k.Sidebar.TabLogs, "logs tab (while open)"),
				helpEntryFrom(k.Sidebar.TabSettings, "settings tab (while open)"),
			},
		},
	}
}

func queryHelpCategories(k KeyMap) []helpCategory {
	return []helpCategory{
		{
			title: "The bar",
			entries: []helpEntry{
				helpEntryFrom(k.Inventory.Search, "focus the query bar"),
				{keys: "type", desc: "results update as you type"},
				{keys: "esc / ⏎", desc: "return focus to the table; the query stays"},
				{keys: "?", desc: "this panel (also while the bar is focused)"},
				{keys: "count", desc: "right side is matching / loaded"},
			},
		},
		{
			title: "Search",
			entries: []helpEntry{
				{keys: "text", desc: "substring AND, case-insensitive"},
				{keys: "haystack", desc: "name, id, provider, region, type, state"},
				{keys: "web prod", desc: "contains web AND contains prod"},
				{keys: `"web api"`, desc: "quoted token keeps spaces"},
				{keys: "us-east", desc: "partial; matches us-east-1"},
			},
		},
		{
			title: "Filters",
			entries: []helpEntry{
				{keys: "field=value", desc: "exact match, case-insensitive"},
				{keys: "field = value", desc: "spaces around = are allowed"},
				{keys: "p=aws s=running", desc: "different fields AND"},
				{keys: "p=aws p=gcp", desc: "same field OR"},
				{keys: "region=us", desc: "does not match us-east-1 (use search)"},
				{keys: "foo=bar", desc: "unknown field is search, not a filter"},
				{keys: "tag=key:value", desc: "exact tag key and value"},
			},
		},
		{
			title: "Sort",
			entries: []helpEntry{
				{keys: "name+", desc: "ascending (preferred)"},
				{keys: "name-", desc: "descending (preferred)"},
				{keys: "+name / -name", desc: "prefix form, also accepted"},
				{keys: "name:asc", desc: "ascending"},
				{keys: "name:desc", desc: "descending"},
				{keys: "last token", desc: "exactly one sort; last sort wins"},
				{keys: "columns", desc: "any field except tag, plus aliases"},
				{keys: "(none)", desc: "keep load order (stable)"},
			},
		},
		{
			title: "Fields",
			entries: []helpEntry{
				{keys: "n / name", desc: "VM name"},
				{keys: "p / provider", desc: "aws, azure, gcp, …"},
				{keys: "r / region", desc: "provider region id"},
				{keys: "t / type", desc: "machine type / size"},
				{keys: "s / state", desc: "running, stopped, starting, stopping, unknown"},
				{keys: "id", desc: "provider instance id"},
				{keys: "tag", desc: "tag=key:value only; not sortable"},
			},
		},
		{
			title: "Shortcuts",
			entries: []helpEntry{
				helpEntryFrom(k.Inventory.Search, "focus query bar"),
				helpEntryFrom(k.Inventory.FilterProv, "cycle provider=, then off"),
				helpEntryFrom(k.Inventory.FilterState, "cycle state= running → … → off"),
				helpEntryFrom(k.Inventory.FilterRegion, "cycle region= in loaded inventory"),
				helpEntryFrom(k.Inventory.SortKey, "cycle sort name → provider → region → type → state → off"),
				helpEntryFrom(k.Inventory.SortDir, "flip sort direction"),
				helpEntryFrom(k.Inventory.ClearFilters, "clear the whole query"),
				{keys: "table", desc: "shortcuts fire only while the table is focused"},
			},
		},
		{
			title: "Palette",
			entries: []helpEntry{
				{keys: ":", desc: "open the command palette"},
				{keys: "aws azure gcp", desc: "set provider="},
				{keys: "all", desc: "clear provider filter"},
				{keys: "running stopped", desc: "set state="},
				{keys: "region", desc: "cycle region= (same as r)"},
				{keys: "clear", desc: "clear the whole query"},
			},
		},
		{
			title: "Examples",
			entries: []helpEntry{
				{keys: "web", desc: "name/id/… contains web"},
				{keys: "p=aws", desc: "provider is aws"},
				{keys: "p=aws web", desc: "aws AND search web"},
				{keys: "provider=aws name-", desc: "aws, sort name descending"},
				{keys: "web p=gcp region+", desc: "search web, gcp, sort region asc"},
				{keys: "name:desc", desc: "sort name descending"},
				{keys: "api p=aws s=running name+", desc: "search, filters, sort name"},
			},
		},
	}
}

func (m helpModel) title() string {
	if m.query {
		return "Filter help"
	}
	return "Help"
}

func (m helpModel) categories(k KeyMap) []helpCategory {
	if m.query {
		return queryHelpCategories(k)
	}
	return helpCategories(k)
}

func (m helpModel) View(s Styles, k KeyMap, w, h int) string {
	cats := m.categories(k)
	stacked := helpShouldStack(s, cats, w)
	body := helpBody(s, cats, stacked)
	bodyLines := strings.Split(body, "\n")
	bodyH := helpBodyHeight(s, h)
	scrollable := len(bodyLines) > bodyH
	scroll := 0

	footer := keyed(k.Nav.Esc, "close")
	if scrollable {
		footer = k.Nav.Up.Keys.Help().Key + " " + keyed(k.Nav.Down, "scroll") + " · " + keyed(k.Nav.Esc, "close")
	}

	title := s.DialogTitle.Render(m.title())
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

func helpShouldStack(s Styles, cats []helpCategory, w int) bool {
	twoCol := helpBody(s, cats, false)
	return lipgloss.Width(twoCol)+scrollbarCols+s.Dialog.GetHorizontalFrameSize() > w
}

func helpBody(s Styles, cats []helpCategory, stacked bool) string {
	if stacked || len(cats) < 2 {
		return renderHelpColumn(s, cats)
	}
	mid := (len(cats) + 1) / 2
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
	cats := m.categories(k)
	lines := lipgloss.Height(helpBody(s, cats, helpShouldStack(s, cats, w)))
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
