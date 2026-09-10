package tui

import (
	"fmt"
	"strings"

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
	ActionSidebar
	ActionQuit
	ActionForceQuit
	ActionReconnect

	// Tabs
	ActionTabInventory
	ActionTabSessions

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
	ActionSidebarNotif
	ActionSidebarLogs
	ActionSidebarSettings
)

// HitID names a clickable region. Fill Binding.Mouse.Hit with the same ID
// the mouse hit-test will return so a click resolves to the same Action
// as the key.
type HitID string

const (
	HitNone HitID = ""

	HitHelp            HitID = "help"
	HitPalette         HitID = "palette"
	HitTheme           HitID = "theme"
	HitLogs            HitID = "logs"
	HitSidebar         HitID = "sidebar"
	HitSidebarNotif    HitID = "sidebar.notifications"
	HitSidebarLogs     HitID = "sidebar.logs"
	HitSidebarSettings HitID = "sidebar.settings"
	HitQuit            HitID = "quit"
	HitReconnect       HitID = "reconnect"
	HitTabInventory    HitID = "tab.inventory"
	HitTabSessions     HitID = "tab.sessions"
	HitJumpTop         HitID = "nav.top"
	HitJumpBottom      HitID = "nav.bottom"
	HitSelect          HitID = "nav.select"
	HitBack            HitID = "nav.back"
	HitSearch          HitID = "inventory.search"
	HitMark            HitID = "inventory.mark"
	HitMarkAll         HitID = "inventory.markAll"
	HitConnect         HitID = "inventory.connect"
	HitStart           HitID = "inventory.start"
	HitStop            HitID = "inventory.stop"
	HitFilterProv      HitID = "inventory.filterProvider"
	HitFilterState     HitID = "inventory.filterState"
	HitFilterRegion    HitID = "inventory.filterRegion"
	HitSortKey         HitID = "inventory.sort"
	HitSortDir         HitID = "inventory.sortDir"
	HitClearFilters    HitID = "inventory.clearFilters"
	HitRefresh         HitID = "inventory.refresh"
	HitCloseSession    HitID = "sessions.close"
	HitQuitSignOut     HitID = "overlay.quit.signOut"
	HitQuitCancel      HitID = "overlay.quit.cancel"
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
		if keyEventMatches(msg, b.Keys) {
			return b.Action
		}
	}
	return ActionNone
}

// keyEventMatches compares bindings against both the printable String()
// form and the modifier+code form. bubbles/key.Matches only uses String(),
// which drops Shift on combos like ctrl+shift+t (it may report "T" or
// "ctrl+T" instead of "ctrl+shift+t").
func keyEventMatches(msg fmt.Stringer, b key.Binding) bool {
	if !b.Enabled() {
		return false
	}
	if key.Matches(msg, b) {
		return true
	}
	got := canonicalEvent(msg)
	if got == "" {
		return false
	}
	for _, want := range b.Keys() {
		if canonicalKeyName(want) == got {
			return true
		}
	}
	return false
}

func canonicalEvent(msg fmt.Stringer) string {
	if kp, ok := msg.(tea.KeyPressMsg); ok {
		if s := canonicalFromKey(kp.Key()); s != "" {
			return s
		}
		return canonicalKeyName(kp.Keystroke())
	}
	return canonicalKeyName(msg.String())
}

func canonicalFromKey(k tea.Key) string {
	hasCtrl := k.Mod.Contains(tea.ModCtrl)
	hasAlt := k.Mod.Contains(tea.ModAlt)
	hasShift := k.Mod.Contains(tea.ModShift)

	code := k.Code
	if k.BaseCode != 0 {
		code = k.BaseCode
	}
	if code >= 'A' && code <= 'Z' {
		hasShift = true
		code = code - 'A' + 'a'
	}

	name := keyCodeName(code)
	if name == "" {
		return ""
	}

	var parts []string
	if hasCtrl {
		parts = append(parts, "ctrl")
	}
	if hasAlt {
		parts = append(parts, "alt")
	}
	if hasShift {
		parts = append(parts, "shift")
	}
	return strings.Join(append(parts, name), "+")
}

func canonicalKeyName(s string) string {
	if s == "" {
		return ""
	}
	parts := strings.Split(s, "+")
	last := parts[len(parts)-1]
	hasShift := false
	mods := make([]string, 0, len(parts))
	for _, p := range parts[:len(parts)-1] {
		p = strings.ToLower(p)
		if p == "shift" {
			hasShift = true
			continue
		}
		mods = append(mods, p)
	}
	if len(last) == 1 && last[0] >= 'A' && last[0] <= 'Z' {
		hasShift = true
	}
	last = strings.ToLower(last)
	if hasShift {
		mods = append(mods, "shift")
	}
	return strings.Join(append(mods, last), "+")
}

func keyCodeName(code rune) string {
	switch code {
	case tea.KeyEnter:
		return "enter"
	case tea.KeyTab:
		return "tab"
	case tea.KeySpace:
		return "space"
	case tea.KeyEsc:
		return "esc"
	case tea.KeyUp:
		return "up"
	case tea.KeyDown:
		return "down"
	case tea.KeyLeft:
		return "left"
	case tea.KeyRight:
		return "right"
	case tea.KeyPgUp:
		return "pgup"
	case tea.KeyPgDown:
		return "pgdown"
	case tea.KeyHome:
		return "home"
	case tea.KeyEnd:
		return "end"
	case tea.KeyBackspace:
		return "backspace"
	default:
		if code > 0 && code < 128 && code != ' ' {
			return strings.ToLower(string(code))
		}
		return ""
	}
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
