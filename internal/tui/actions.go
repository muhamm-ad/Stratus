package tui

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// Action is a named user intent. Keyboard and mouse both resolve to an
// Action so handlers never switch on raw key or click strings.
type Action int

const (
	ActionNone Action = iota

	// Global
	ActionHelp
	ActionPalette
	ActionTheme
	ActionLogs
	ActionQuit
	ActionForceQuit
	ActionReconnect

	// Tabs
	ActionTabInventory
	ActionTabSessions
	ActionTabAudit
	ActionTabSettings

	// Navigation (shared across tabs and overlays)
	ActionMoveUp
	ActionMoveDown
	ActionJumpTop
	ActionJumpBottom
	ActionSelect
	ActionBack
	ActionPageUp
	ActionPageDown

	// Inventory
	ActionSearch
	ActionMark
	ActionMarkAll
	ActionConnect
	ActionStart
	ActionStop
	ActionFilterProvider
	ActionFilterState
	ActionFilterRegion
	ActionSortKey
	ActionSortDir
	ActionClearFilters
	ActionRefresh

	// Sessions
	ActionCloseSession

	// Overlays
	ActionOverlayNext
	ActionOverlayPrev
	ActionOverlayConfirm
	ActionOverlayYes
	ActionOverlayNo
)

// HitID names a clickable region. Fill Binding.Mouse.Hit with the same ID
// the mouse hit-test will return so a click resolves to the same Action
// as the key.
type HitID string

const (
	HitNone HitID = ""

	HitHelp         HitID = "help"
	HitPalette      HitID = "palette"
	HitTheme        HitID = "theme"
	HitLogs         HitID = "logs"
	HitQuit         HitID = "quit"
	HitReconnect    HitID = "reconnect"
	HitTabInventory HitID = "tab.inventory"
	HitTabSessions  HitID = "tab.sessions"
	HitTabAudit     HitID = "tab.audit"
	HitTabSettings  HitID = "tab.settings"
	HitJumpTop      HitID = "nav.top"
	HitJumpBottom   HitID = "nav.bottom"
	HitSelect       HitID = "nav.select"
	HitBack         HitID = "nav.back"
	HitSearch       HitID = "inventory.search"
	HitMark         HitID = "inventory.mark"
	HitMarkAll      HitID = "inventory.markAll"
	HitConnect      HitID = "inventory.connect"
	HitStart        HitID = "inventory.start"
	HitStop         HitID = "inventory.stop"
	HitFilterProv   HitID = "inventory.filterProvider"
	HitFilterState  HitID = "inventory.filterState"
	HitFilterRegion HitID = "inventory.filterRegion"
	HitSortKey      HitID = "inventory.sort"
	HitSortDir      HitID = "inventory.sortDir"
	HitClearFilters HitID = "inventory.clearFilters"
	HitRefresh      HitID = "inventory.refresh"
	HitCloseSession HitID = "sessions.close"
	HitQuitSignOut  HitID = "overlay.quit.signOut"
	HitQuitCancel   HitID = "overlay.quit.cancel"
)

// MouseBinding is the click/wheel counterpart of a key binding.
// A zero value means the action has no mouse trigger yet.
//
//   - Button + Hit: click on a named region (tabs, buttons, row actions)
//   - Button only:   global pointer gesture (wheel → move)
type MouseBinding struct {
	Button tea.MouseButton
	Hit    HitID
}

// Matches reports whether this mouse binding is the one that fired.
func (m MouseBinding) Matches(button tea.MouseButton, hit HitID) bool {
	if m.Button == tea.MouseNone {
		return false
	}
	if m.Button != button {
		return false
	}
	if m.Hit == HitNone {
		return true
	}
	return m.Hit == hit
}

// Binding pairs one Action with its keyboard keys and (optional) mouse
// gesture. Adding mouse later is filling Mouse and calling MatchMouse —
// dispatch stays on Action.
type Binding struct {
	Action Action
	Keys   key.Binding
	Mouse  MouseBinding
}

func bind(act Action, hit HitID, helpKey, helpDesc string, keys ...string) Binding {
	b := Binding{
		Action: act,
		Keys:   key.NewBinding(key.WithKeys(keys...), key.WithHelp(helpKey, helpDesc)),
	}
	if hit != HitNone {
		b.Mouse = MouseBinding{Button: tea.MouseLeft, Hit: hit}
	}
	return b
}

func bindWheel(act Action, button tea.MouseButton, helpKey, helpDesc string, keys ...string) Binding {
	return Binding{
		Action: act,
		Keys:   key.NewBinding(key.WithKeys(keys...), key.WithHelp(helpKey, helpDesc)),
		Mouse:  MouseBinding{Button: button},
	}
}

func matchKey(msg fmt.Stringer, bs []Binding) Action {
	for _, b := range bs {
		if key.Matches(msg, b.Keys) {
			return b.Action
		}
	}
	return ActionNone
}

func matchMouse(button tea.MouseButton, hit HitID, bs []Binding) Action {
	for _, b := range bs {
		if b.Mouse.Matches(button, hit) {
			return b.Action
		}
	}
	return ActionNone
}

func keyed(b Binding, desc string) string {
	k := b.Keys.Help().Key
	if desc == "" {
		return k
	}
	return k + " " + desc
}
