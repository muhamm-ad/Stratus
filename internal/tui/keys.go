package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// KeyMap groups bindings by scope so global shortcuts, navigation, and
// each tab stay independent. Mouse hit-testing later calls MatchMouse
// with the same scopes.
type KeyMap struct {
	Global    GlobalMap
	Nav       NavMap
	Inventory InventoryMap
	Sessions  SessionsMap
	Audit     AuditMap
	Settings  SettingsMap
	Overlay   OverlayMap
	Sidebar   SidebarMap
}

func DefaultKeys() KeyMap {
	return KeyMap{
		Global:    defaultGlobalKeys(),
		Nav:       defaultNavKeys(),
		Inventory: defaultInventoryKeys(),
		Sessions:  defaultSessionsKeys(),
		Audit:     defaultAuditKeys(),
		Settings:  defaultSettingsKeys(),
		Overlay:   defaultOverlayKeys(),
		Sidebar:   defaultSidebarKeys(),
	}
}

func (k KeyMap) MatchTab(msg fmt.Stringer, t tab) Action {
	switch t {
	case tabInventory:
		return k.Inventory.Match(msg)
	case tabSessions:
		return k.Sessions.Match(msg)
	case tabAudit:
		return k.Audit.Match(msg)
	case tabSettings:
		return k.Settings.Match(msg)
	default:
		return ActionNone
	}
}

// MatchMouse resolves a pointer event to the same Action MatchKey would.
// Pass the hit-tested region (or HitNone for wheel / untargeted gestures).
func (k KeyMap) MatchMouse(button tea.MouseButton, hit HitID, t tab) Action {
	if act := k.Global.MatchMouse(button, hit); act != ActionNone {
		return act
	}
	if act := k.Nav.MatchMouse(button, hit); act != ActionNone {
		return act
	}
	if act := k.Overlay.MatchMouse(button, hit); act != ActionNone {
		return act
	}
	if act := k.Sidebar.MatchMouse(button, hit); act != ActionNone {
		return act
	}
	switch t {
	case tabInventory:
		return k.Inventory.MatchMouse(button, hit)
	case tabSessions:
		return k.Sessions.MatchMouse(button, hit)
	case tabAudit:
		return k.Audit.MatchMouse(button, hit)
	case tabSettings:
		return k.Settings.MatchMouse(button, hit)
	default:
		return ActionNone
	}
}

func (k KeyMap) StatusHint(t tab) string {
	result := []string{}
	// switch t {
	// case tabInventory:
	// 	result = strings.Join([]string{
	// 		keyed(k.Nav.Enter, "detail"),
	// 		keyed(k.Inventory.Mark, "mark"),
	// 		keyed(k.Inventory.Connect, "connect"),
	// 		keyed(k.Inventory.Search, "filter"),
	// 		keyed(k.Inventory.SortKey, "sort"),
	// 	}, " · ")
	// case tabSessions:
	// 	result = append(result,
	// 		keyed(k.Nav.Enter, "jump to vm"),
	// 	)
	// case tabAudit:
	// 	result = strings.Join([]string{
	// 		keyed(k.Audit.Search, "filter"),
	// 	}, " · ")
	// default:
	// 	result = []string{}
	// }

	globalHints := []string{
		keyed(k.Global.Palette, "cmd"),
		keyed(k.Global.Help, "help"),
		keyed(k.Global.Sidebar, "sidebar"),
	}
	result = append(result, globalHints...)

	return strings.Join(result, " · ")
}

// ---------------------------------------------------------------------------
// Global
// ---------------------------------------------------------------------------

// GlobalMap is always active on the app screen (not while typing search).
type GlobalMap struct {
	Help, Palette, Theme, Logs Binding
	Sidebar                    Binding
	Quit, ForceQuit            Binding
	Reconnect                  Binding
	TabInventory, TabSessions  Binding
	TabAudit, TabSettings      Binding
}

func defaultGlobalKeys() GlobalMap {
	return GlobalMap{
		Help:         bind(ActionHelp, HitHelp, "ctrl+h/?", "help", "ctrl+h", "?"),
		Palette:      bind(ActionPalette, HitPalette, ":", "cmd", ":"),
		Theme:        bind(ActionTheme, HitTheme, "ctrl+shift+t", "theme", "ctrl+shift+t"),
		Logs:         bind(ActionLogs, HitLogs, "ctrl+shift+l", "log alerts", "ctrl+shift+l"),
		Sidebar:      bind(ActionSidebar, HitSidebar, "ctrl+r", "sidebar", "ctrl+r"),
		Quit:         bind(ActionQuit, HitQuit, "ctrl+q", "quit", "ctrl+q"),
		ForceQuit:    bind(ActionForceQuit, HitNone, "ctrl+c", "force quit", "ctrl+c"),
		Reconnect:    bind(ActionReconnect, HitReconnect, "R", "reconnect", "R"),
		TabInventory: bind(ActionTabInventory, HitTabInventory, "1", "inventory", "1"),
		TabSessions:  bind(ActionTabSessions, HitTabSessions, "2", "sessions", "2"),
		TabAudit:     bind(ActionTabAudit, HitTabAudit, "3", "audit", "3"),
		TabSettings:  bind(ActionTabSettings, HitTabSettings, "4", "settings", "4"),
	}
}

func (g GlobalMap) Bindings() []Binding {
	return []Binding{
		g.ForceQuit, g.Help, g.Palette, g.Theme, g.Logs, g.Sidebar, g.Quit, g.Reconnect,
		g.TabInventory, g.TabSessions, g.TabAudit, g.TabSettings,
	}
}

func (g GlobalMap) Match(msg fmt.Stringer) Action {
	return matchKey(msg, g.Bindings())
}

func (g GlobalMap) MatchMouse(button tea.MouseButton, hit HitID) Action {
	return matchMouse(button, hit, g.Bindings())
}

// ---------------------------------------------------------------------------
// Navigation
// ---------------------------------------------------------------------------

// NavMap is shared list/table movement: keys today, wheel/click later.
type NavMap struct {
	Up, Down         Binding
	Top, Bottom      Binding
	Enter, Esc       Binding
	PageUp, PageDown Binding
}

func defaultNavKeys() NavMap {
	return NavMap{
		Up:       bindWheel(ActionMoveUp, tea.MouseWheelUp, "↑/k", "move", "k", "up"),
		Down:     bindWheel(ActionMoveDown, tea.MouseWheelDown, "↓/j", "move", "j", "down"),
		Top:      bind(ActionJumpTop, HitJumpTop, "gg", "top", "g", "home"),
		Bottom:   bind(ActionJumpBottom, HitJumpBottom, "G", "bottom", "G", "end"),
		Enter:    bind(ActionSelect, HitSelect, "⏎", "select", "enter"),
		Esc:      bind(ActionBack, HitBack, "esc", "close", "esc"),
		PageUp:   bind(ActionPageUp, HitNone, "pgup", "page up", "pgup"),
		PageDown: bind(ActionPageDown, HitNone, "pgdn", "page down", "pgdown"),
	}
}

func (n NavMap) Bindings() []Binding {
	return []Binding{n.Up, n.Down, n.Top, n.Bottom, n.Enter, n.Esc, n.PageUp, n.PageDown}
}

func (n NavMap) Match(msg fmt.Stringer) Action {
	return matchKey(msg, n.Bindings())
}

func (n NavMap) MatchMouse(button tea.MouseButton, hit HitID) Action {
	return matchMouse(button, hit, n.Bindings())
}

// ---------------------------------------------------------------------------
// Overlay
// ---------------------------------------------------------------------------

// OverlayMap is modal chrome (help, palette, quit confirm).
type OverlayMap struct {
	Next, Prev Binding
	Confirm    Binding
	Yes, No    Binding
}

func defaultOverlayKeys() OverlayMap {
	return OverlayMap{
		Next:    bind(ActionOverlayNext, HitNone, "tab", "next button", "tab", "right"),
		Prev:    bind(ActionOverlayPrev, HitNone, "shift+tab", "prev button", "shift+tab", "left"),
		Confirm: bind(ActionOverlayConfirm, HitNone, "⏎", "confirm", "enter", "space"),
		Yes:     bind(ActionOverlayYes, HitQuitSignOut, "y", "sign out", "y"),
		No:      bind(ActionOverlayNo, HitQuitCancel, "n", "cancel", "n"),
	}
}

func (o OverlayMap) Bindings() []Binding {
	return []Binding{o.Next, o.Prev, o.Confirm, o.Yes, o.No}
}

func (o OverlayMap) Match(msg fmt.Stringer) Action {
	return matchKey(msg, o.Bindings())
}

func (o OverlayMap) MatchMouse(button tea.MouseButton, hit HitID) Action {
	return matchMouse(button, hit, o.Bindings())
}

// ---------------------------------------------------------------------------
// Sidebar
// ---------------------------------------------------------------------------

// SidebarMap is only matched while the sidebar overlay is open, so 1/2
// keep switching the main tabs when the drawer is closed.
type SidebarMap struct {
	TabNotif, TabLogs Binding
}

func defaultSidebarKeys() SidebarMap {
	return SidebarMap{
		TabNotif: bind(ActionSidebarNotif, HitSidebarNotif, "n", "notifications", "n"),
		TabLogs:  bind(ActionSidebarLogs, HitSidebarLogs, "l", "logs", "l"),
	}
}

func (s SidebarMap) Bindings() []Binding {
	return []Binding{s.TabNotif, s.TabLogs}
}

func (s SidebarMap) Match(msg fmt.Stringer) Action {
	return matchKey(msg, s.Bindings())
}

func (s SidebarMap) MatchMouse(button tea.MouseButton, hit HitID) Action {
	return matchMouse(button, hit, s.Bindings())
}

// ---------------------------------------------------------------------------
// Inventory
// ---------------------------------------------------------------------------

// InventoryMap is only matched on the inventory tab.
type InventoryMap struct {
	Search, Mark, MarkAll                 Binding
	Connect, Start, Stop                  Binding
	FilterProv, FilterState, FilterRegion Binding
	SortKey, SortDir, ClearFilters        Binding
	Refresh                               Binding
}

func defaultInventoryKeys() InventoryMap {
	return InventoryMap{
		Search:       bind(ActionSearch, HitSearch, "/", "filter", "/"),
		Mark:         bind(ActionMark, HitMark, "s", "mark", "s"),
		MarkAll:      bind(ActionMarkAll, HitMarkAll, "a", "mark all/none", "a"),
		Connect:      bind(ActionConnect, HitConnect, "c", "connect", "c"),
		Start:        bind(ActionStart, HitStart, "ctrl+r", "start", "ctrl+r"),
		Stop:         bind(ActionStop, HitStop, "ctrl+s", "stop", "ctrl+s"),
		FilterProv:   bind(ActionFilterProvider, HitFilterProv, "p", "provider", "p"),
		FilterState:  bind(ActionFilterState, HitFilterState, "f", "state", "f"),
		FilterRegion: bind(ActionFilterRegion, HitFilterRegion, "r", "region", "r"),
		SortKey:      bind(ActionSortKey, HitSortKey, "o", "sort", "o"),
		SortDir:      bind(ActionSortDir, HitSortDir, "O", "sort dir", "O"),
		ClearFilters: bind(ActionClearFilters, HitClearFilters, "x", "clear filters", "x"),
		Refresh:      bind(ActionRefresh, HitRefresh, "u", "refresh", "u"),
	}
}

func (m InventoryMap) Bindings() []Binding {
	return []Binding{
		m.Search, m.Mark, m.MarkAll, m.Connect, m.Start, m.Stop,
		m.FilterProv, m.FilterState, m.FilterRegion,
		m.SortKey, m.SortDir, m.ClearFilters, m.Refresh,
	}
}

func (m InventoryMap) Match(msg fmt.Stringer) Action {
	return matchKey(msg, m.Bindings())
}

func (m InventoryMap) MatchMouse(button tea.MouseButton, hit HitID) Action {
	return matchMouse(button, hit, m.Bindings())
}

// ---------------------------------------------------------------------------
// Sessions
// ---------------------------------------------------------------------------

// SessionsMap is only matched on the sessions tab.
type SessionsMap struct {
	Close Binding
}

func defaultSessionsKeys() SessionsMap {
	return SessionsMap{
		Close: bind(ActionCloseSession, HitCloseSession, "x", "close session", "x", "d"),
	}
}

func (m SessionsMap) Bindings() []Binding {
	return []Binding{m.Close}
}

func (m SessionsMap) Match(msg fmt.Stringer) Action {
	return matchKey(msg, m.Bindings())
}

func (m SessionsMap) MatchMouse(button tea.MouseButton, hit HitID) Action {
	return matchMouse(button, hit, m.Bindings())
}

// ---------------------------------------------------------------------------
// Audit
// ---------------------------------------------------------------------------

// AuditMap is a placeholder so audit-only shortcuts have a home.
type AuditMap struct{}

func defaultAuditKeys() AuditMap { return AuditMap{} }

func (m AuditMap) Bindings() []Binding { return nil }

func (m AuditMap) Match(msg fmt.Stringer) Action { return ActionNone }

func (m AuditMap) MatchMouse(button tea.MouseButton, hit HitID) Action {
	return ActionNone
}

// ---------------------------------------------------------------------------
// Settings
// ---------------------------------------------------------------------------

// SettingsMap is a placeholder so settings-only shortcuts have a home.
type SettingsMap struct{}

func defaultSettingsKeys() SettingsMap { return SettingsMap{} }

func (m SettingsMap) Bindings() []Binding { return nil }

func (m SettingsMap) Match(msg fmt.Stringer) Action { return ActionNone }

func (m SettingsMap) MatchMouse(button tea.MouseButton, hit HitID) Action {
	return ActionNone
}
