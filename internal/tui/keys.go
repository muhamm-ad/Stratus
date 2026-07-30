package tui

import "charm.land/bubbles/v2/key"

// KeyMap holds all bindings. Help text strings mirror the mockup exactly.
type KeyMap struct {
	// navigation
	Tab1, Tab2, Tab3, Tab4 key.Binding
	Up, Down, Top, Bottom  key.Binding
	Enter, Esc             key.Binding
	// global
	Search, Cmd, Logs, Theme, Help, Quit key.Binding
	// inventory
	Mark, MarkAll, Connect, Start, Stop          key.Binding
	FilterProv, FilterState, FilterRegion key.Binding
	SortKey, SortDir, ClearFilters        key.Binding
	Reconnect, Refresh                    key.Binding
	// sessions
	CloseSess key.Binding
}

func DefaultKeys() KeyMap {
	return KeyMap{
		Tab1:         key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "inventory")),
		Tab2:         key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "sessions")),
		Tab3:         key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "audit")),
		Tab4:         key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "settings")),
		Up:           key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("↑/k", "move")),
		Down:         key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("↓/j", "move")),
		Top:          key.NewBinding(key.WithKeys("g"), key.WithHelp("gg", "top")),
		Bottom:       key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "bottom")),
		Enter:        key.NewBinding(key.WithKeys("enter"), key.WithHelp("⏎", "detail")),
		Esc:          key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close")),
		Search:       key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Cmd:          key.NewBinding(key.WithKeys(":"), key.WithHelp(":", "cmd")),
		Logs:         key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "logs")),
		Theme:        key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "theme")),
		Help:         key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:         key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		Mark:         key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "mark")),
		MarkAll:      key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "mark all/none")),
		Connect:      key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "connect")),
		Start:        key.NewBinding(key.WithKeys("ctrl+r"), key.WithHelp("ctrl+r", "start")),
		Stop:         key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "stop")),
		FilterProv:   key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "provider")),
		FilterState:  key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "state")),
		FilterRegion: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "region")),
		SortKey:      key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "sort")),
		SortDir:      key.NewBinding(key.WithKeys("O"), key.WithHelp("O", "sort dir")),
		ClearFilters: key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "clear filters")),
		Reconnect:    key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "reconnect")),
		Refresh:      key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "refresh")),
		CloseSess:    key.NewBinding(key.WithKeys("x", "d"), key.WithHelp("x", "close session")),
	}
}

// Contextual hint strings, taken verbatim from the mockup.
const (
	HintInventory = "⏎ detail · s mark · c connect · / filter · o sort · : cmd · ? help"
	HintSessions  = "⏎ jump to vm · x close · : cmd · ? help"
	HintAudit     = "/ filter · : cmd · ? help"
	HintSettings  = "⏎ toggle/cycle · : cmd · ? help"
)

// ShortHelp and FullHelp satisfy bubbles/v2/help's KeyMap interface, so
// help.Model can render this same KeyMap directly instead of the app keeping
// a second, hand-written copy of every binding's help text.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Cmd, k.Help, k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Tab1, k.Tab2, k.Tab3, k.Tab4, k.Up, k.Down, k.Top, k.Bottom},
		{k.Enter, k.Mark, k.MarkAll, k.Connect, k.Start, k.Stop, k.Search,
			k.SortKey, k.SortDir, k.FilterProv, k.FilterState, k.FilterRegion, k.ClearFilters, k.Refresh},
		{k.CloseSess},
		{k.Cmd, k.Logs, k.Theme, k.Help, k.Reconnect, k.Quit},
	}
}
