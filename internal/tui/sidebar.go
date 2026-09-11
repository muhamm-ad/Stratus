package tui

import (
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"go.dalton.dog/bubbleup/v2"
)

type sidebarTab int

const (
	sidebarTabNotif sidebarTab = iota
	sidebarTabLogs
	sidebarTabSettings
)

// sidebarWidth is fixed so the drawer never shrinks (and wraps) on resize.
const sidebarWidth = 56

// notifIconGap sits between the status glyph and the message.
const notifIconGap = "  "

type notifEntry struct {
	ts, key, msg string
	unread       bool
}

type notifPane struct {
	entries []notifEntry
}

// add records a notification. It returns false when the same message is
// already in the list so callers can skip a repeat toast.
func (n *notifPane) add(key, msg string) bool {
	now := time.Now().Format("15:04:05")
	for i, e := range n.entries {
		if e.msg == msg {
			e.ts = now
			e.key = key
			e.unread = true
			n.entries = append(n.entries[:i], n.entries[i+1:]...)
			n.entries = append([]notifEntry{e}, n.entries...)
			return false
		}
	}
	n.entries = append([]notifEntry{{ts: now, key: key, msg: msg, unread: true}}, n.entries...)
	if len(n.entries) > maxLogEntries {
		n.entries = n.entries[:maxLogEntries]
	}
	return true
}

func (n *notifPane) unreadCount() int {
	c := 0
	for _, e := range n.entries {
		if e.unread {
			c++
		}
	}
	return c
}

func (n *notifPane) markAllRead() {
	for i := range n.entries {
		n.entries[i].unread = false
	}
}

type sidebarModel struct {
	tab    sidebarTab
	scroll int
	notifs notifPane
}

func newSidebarModel() sidebarModel { return sidebarModel{} }

func sidebarHandleView(s Styles) string {
	return s.TabInactive.Align(lipgloss.Center).Render("│◀")
}

func (m *sidebarModel) setTab(t sidebarTab) {
	if m.tab != t {
		m.tab = t
		m.scroll = 0
	}
}

func (m sidebarModel) View(s Styles, k KeyMap, h int, logs logPane, settings settingsModel, themeIdx int) string {
	h = max(1, h)
	panelW := sidebarWidth

	innerW := sidebarInnerWidth(s)
	innerH := max(1, h-s.Sidebar.GetVerticalFrameSize())

	tabs := m.tabRow(s, innerW)
	footer := sidebarFooter(s, k, innerW)
	listH := max(1, innerH-blockHeight(tabs)-blockHeight(footer))

	body := m.listBody(s, logs, settings, themeIdx, innerW)
	bodyLines := strings.Split(body, "\n")
	if body == "" {
		bodyLines = nil
	}
	n := len(bodyLines)
	if n > listH {
		maxScroll := max(0, n-listH)
		scroll := max(0, min(m.scroll, maxScroll))
		end := min(scroll+listH, n)
		body = joinScrollbar(strings.Join(bodyLines[scroll:end], "\n"), innerW, listH, n, scroll, s)
	} else if n == 0 {
		empty := "no notifications yet"
		if m.tab == sidebarTabLogs {
			empty = "no logs yet"
		}
		body = s.DialogKey.Render(empty)
	}

	inner := stackTopMidBottom(tabs, body, footer, innerW, innerH)
	return boxNoWrap(s.Sidebar.PaddingBottom(0).MarginBottom(0), inner, panelW, h)
}

func (m sidebarModel) tabRow(s Styles, innerW int) string {
	notif := sidebarTabStyle(s, m.tab == sidebarTabNotif).Render("notifications [n]")
	logs := sidebarTabStyle(s, m.tab == sidebarTabLogs).Render("logs [l]")
	settings := sidebarTabStyle(s, m.tab == sidebarTabSettings).Render("settings [s]")
	if lipgloss.Width(notif)+lipgloss.Width(logs)+lipgloss.Width(settings) > innerW {
		top := sidebarTabFill(s, innerW, notif, logs)
		bottom := sidebarTabFill(s, innerW, settings)
		row := lipgloss.JoinVertical(lipgloss.Left, top, bottom)
		return padBlock(row, innerW, max(1, lipgloss.Height(row)))
	}
	return sidebarTabFill(s, innerW, notif, logs, settings)
}

func sidebarTabFill(s Styles, innerW int, tabs ...string) string {
	used := 0
	for _, t := range tabs {
		used += lipgloss.Width(t)
	}
	parts := append([]string{}, tabs...)
	if fillW := max(0, innerW-used); fillW > 0 {
		fill := s.TabInactive.
			BorderTop(false).
			BorderLeft(false).
			BorderRight(false).
			Padding(0, 0).
			Width(fillW).
			MaxWidth(fillW).
			Render("")
		parts = append(parts, fill)
	}
	row := lipgloss.JoinHorizontal(lipgloss.Bottom, parts...)
	return padBlock(row, innerW, max(1, lipgloss.Height(row)))
}

func sidebarFooter(s Styles, k KeyMap, innerW int) string {
	hint := keyed(k.Nav.Esc, "") + " or " + keyed(k.Global.Sidebar, "") + " to hide"
	return s.DialogKey.Padding(0).Margin(0).Render(clipLine(hint, innerW))
}

func sidebarTabStyle(s Styles, active bool) lipgloss.Style {
	if active {
		return s.TabActive
	}
	return s.TabInactive
}

func (m sidebarModel) listBody(s Styles, logs logPane, settings settingsModel, themeIdx, innerW int) string {
	switch m.tab {
	case sidebarTabLogs:
		if len(logs.entries) == 0 {
			return ""
		}
		rows := make([]string, 0, len(logs.entries))
		for i := len(logs.entries) - 1; i >= 0; i-- {
			e := logs.entries[i]
			rows = append(rows, clipLine(formatLogRow(s, e), innerW))
		}
		return strings.Join(rows, "\n")
	case sidebarTabSettings:
		return settings.sidebarBody(s, innerW, themeIdx)
	default:
		if len(m.notifs.entries) == 0 {
			return ""
		}
		rows := make([]string, 0, len(m.notifs.entries))
		for _, e := range m.notifs.entries {
			rows = append(rows, formatNotifBlock(s, e, innerW))
		}
		// rule := s.Dim.Render(strings.Repeat("─", max(1, innerW)))
		// return strings.Join(rows, "\n"+rule+"\n")
		return strings.Join(rows, "\n")
	}
}

func formatNotifBlock(s Styles, e notifEntry, innerW int) string {
	msgStyle := s.OK
	switch e.key {
	case bubbleup.ErrorKey:
		msgStyle = s.Err
	case bubbleup.WarnKey:
		msgStyle = s.Warn
	case bubbleup.DebugKey:
		msgStyle = s.Dim
	}

	prefix := s.Dim.Render(e.ts) + " "
	prefixW := lipgloss.Width(prefix)
	msgW := max(4, innerW-prefixW)
	wrapped := lipgloss.Wrap(e.msg, msgW, "")
	indent := strings.Repeat(" ", prefixW)
	lines := strings.Split(wrapped, "\n")
	out := make([]string, len(lines))
	for i, line := range lines {
		msg := msgStyle.Render(line)
		if i == 0 {
			out[i] = prefix + msg
			continue
		}
		out[i] = indent + msg
	}
	return strings.Join(out, "\n")
}

func formatLogRow(s Styles, e logEntry) string {
	lvl := s.Dim
	switch e.level {
	case "WARN":
		lvl = s.Warn
	case "ERR", "ERROR":
		lvl = s.Err
	case "INFO":
		lvl = s.Cyan
	}
	return s.Dim.Render(e.ts) + " " + lvl.Render(e.level) + "  " + s.Text.Render(e.msg)
}

func sidebarInnerWidth(s Styles) int {
	return max(8, sidebarWidth-s.Sidebar.GetHorizontalFrameSize())
}

func (m sidebarModel) listHeight(s Styles, h int) int {
	innerW := sidebarInnerWidth(s)
	innerH := max(1, max(1, h)-s.Sidebar.GetVerticalFrameSize())
	footerH := blockHeight(s.DialogKey.Render("hint"))
	return max(1, innerH-blockHeight(m.tabRow(s, innerW))-footerH)
}

func (m sidebarModel) bodyLineCount(s Styles, logs logPane, settings settingsModel, themeIdx int) int {
	body := m.listBody(s, logs, settings, themeIdx, sidebarInnerWidth(s))
	if body == "" {
		return 0
	}
	return lipgloss.Height(body)
}

func (m *sidebarModel) maxScroll(s Styles, h int, logs logPane, settings settingsModel, themeIdx int) int {
	return max(0, m.bodyLineCount(s, logs, settings, themeIdx)-m.listHeight(s, h))
}

func (m *sidebarModel) scrollBy(delta int, s Styles, h int, logs logPane, settings settingsModel, themeIdx int) {
	m.scroll += delta
	maxScroll := m.maxScroll(s, h, logs, settings, themeIdx)
	if m.scroll < 0 {
		m.scroll = 0
	}
	if m.scroll > maxScroll {
		m.scroll = maxScroll
	}
}

func (m *sidebarModel) pageSize(s Styles, h int) int {
	return m.listHeight(s, h)
}
