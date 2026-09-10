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
)

// sidebarWidth is fixed so the drawer never shrinks (and wraps) on resize.
const sidebarWidth = 56

// notifIconGap sits between the status glyph and the message.
const notifIconGap = "  "

type notifEntry struct {
	ts, key, msg string
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
			n.entries = append(n.entries[:i], n.entries[i+1:]...)
			n.entries = append([]notifEntry{e}, n.entries...)
			return false
		}
	}
	n.entries = append([]notifEntry{{ts: now, key: key, msg: msg}}, n.entries...)
	if len(n.entries) > maxLogEntries {
		n.entries = n.entries[:maxLogEntries]
	}
	return true
}

type sidebarModel struct {
	tab    sidebarTab
	scroll int
	notifs notifPane
}

func newSidebarModel() sidebarModel { return sidebarModel{} }

func sidebarHandleView(s Styles, open bool) string {
	glyph := "◀"
	if open {
		glyph = "▶"
	}
	return s.Handle.Render(glyph)
}

func (m *sidebarModel) setTab(t sidebarTab) {
	if m.tab != t {
		m.tab = t
		m.scroll = 0
	}
}

func (m sidebarModel) View(s Styles, k KeyMap, h int, logs logPane) string {
	h = max(1, h)
	panelW := sidebarWidth

	innerW := sidebarInnerWidth(s)
	innerH := max(1, h-s.Sidebar.GetVerticalFrameSize())

	tabs := m.tabRow(s, innerW)
	footer := sidebarFooter(s, k, innerW)
	listH := max(1, innerH-lipgloss.Height(tabs)-lipgloss.Height(footer))

	body := m.listBody(s, logs, innerW)
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
		body = padBlock(s.DialogKey.Render(empty), innerW, listH)
	} else {
		body = padBlock(body, innerW, listH)
	}

	inner := lipgloss.JoinVertical(lipgloss.Left, tabs, body, footer)
	return boxNoWrap(s.Sidebar, inner, panelW, h)
}

func (m sidebarModel) tabRow(s Styles, innerW int) string {
	notif := sidebarTabStyle(s, m.tab == sidebarTabNotif).Render("notifications [n]")
	logs := sidebarTabStyle(s, m.tab == sidebarTabLogs).Render("logs [l]")
	used := lipgloss.Width(notif) + lipgloss.Width(logs)
	fillW := max(0, innerW-used)
	parts := []string{notif, logs}
	if fillW > 0 {
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
	return s.DialogKey.PaddingTop(1).Render(clipLine(hint, innerW))
}

func sidebarTabStyle(s Styles, active bool) lipgloss.Style {
	if active {
		return s.TabActive
	}
	return s.TabInactive
}

func (m sidebarModel) listBody(s Styles, logs logPane, innerW int) string {
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
	default:
		if len(m.notifs.entries) == 0 {
			return ""
		}
		rows := make([]string, 0, len(m.notifs.entries))
		for _, e := range m.notifs.entries {
			rows = append(rows, formatNotifBlock(s, e, innerW))
		}
		return strings.Join(rows, "\n\n")
	}
}

func formatNotifBlock(s Styles, e notifEntry, innerW int) string {
	mark, msgStyle := "●", s.OK
	switch e.key {
	case bubbleup.ErrorKey:
		mark, msgStyle = "✖", s.Err
	case bubbleup.WarnKey:
		mark, msgStyle = "▲", s.Warn
	case bubbleup.DebugKey:
		mark, msgStyle = "◆", s.Dim
	}
	prefix := s.Dim.Render(e.ts) + " " + msgStyle.Render(mark) + notifIconGap
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
	footerH := lipgloss.Height(s.DialogKey.PaddingTop(1).Render("hint"))
	return max(1, innerH-lipgloss.Height(m.tabRow(s, innerW))-footerH)
}

func (m sidebarModel) bodyLineCount(s Styles, logs logPane) int {
	body := m.listBody(s, logs, sidebarInnerWidth(s))
	if body == "" {
		return 0
	}
	return lipgloss.Height(body)
}

func (m *sidebarModel) maxScroll(s Styles, h int, logs logPane) int {
	return max(0, m.bodyLineCount(s, logs)-m.listHeight(s, h))
}

func (m *sidebarModel) scrollBy(delta int, s Styles, h int, logs logPane) {
	m.scroll += delta
	maxScroll := m.maxScroll(s, h, logs)
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
