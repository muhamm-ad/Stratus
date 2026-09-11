package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/muhamm-ad/stratus/internal/core"
)

func (a *App) propagateSize() {
	h := a.contentHeight()
	w := a.layoutWidth()
	a.inv.SetSize(w, h)
	a.sess.SetSize(w, h)
}

func (a *App) contentHeight() int {
	h := a.layoutHeight() - blockHeight(a.topChromeView()) - blockHeight(a.bottomChromeView())
	if h < 1 {
		return 1
	}
	return h
}

func (a *App) topChromeView() string {
	tabBar := a.tabBarView()
	w := a.layoutWidth()
	gapWidth := max(0, w-lipgloss.Width(tabBar))
	tabH := max(1, lipgloss.Height(tabBar))
	if gapWidth == 0 {
		return boxNoWrap(a.styles.Header, tabBar, w, tabH)
	}

	handle := ""
	if a.overlay != overlaySidebar {
		handle = sidebarHandleView(a.styles)
	}
	cluster := handle
	if lipgloss.Width(cluster) > gapWidth {
		cluster = lipgloss.NewStyle().MaxWidth(gapWidth).MaxHeight(tabH).Render(cluster)
	}

	fillW := max(0, gapWidth-lipgloss.Width(cluster))
	if fillW == 0 {
		row := lipgloss.JoinHorizontal(lipgloss.Bottom, tabBar, cluster)
		return boxNoWrap(a.styles.Header, row, w, tabH)
	}
	blank := strings.Repeat(" ", fillW)
	fillLines := make([]string, tabH)
	for i := range fillLines {
		fillLines[i] = blank
	}
	fillLines[tabH-1] = lipgloss.NewStyle().Foreground(a.styles.th.Border).Render(strings.Repeat("─", fillW))
	gap := lipgloss.JoinHorizontal(lipgloss.Bottom, lipgloss.JoinVertical(lipgloss.Left, fillLines...), cluster)
	row := lipgloss.JoinHorizontal(lipgloss.Bottom, tabBar, gap)
	return boxNoWrap(a.styles.Header, row, w, tabH)
}

func (a *App) providerPillsView() string {
	ids := a.svc.GetCloudProvidersIDs()
	pills := make([]string, 0, len(ids))
	for _, cp := range ids {
		var glyph string
		switch a.svc.GetCloudProviderStatus(cp) {
		case core.CloudProviderStatusAuthenticated:
			glyph = "✓"
		case core.CloudProviderStatusAuthenticating:
			glyph = SpinnerFrames[0]
		case core.CloudProviderStatusError:
			glyph = "!"
		default:
			glyph = "?"
		}
		pills = append(pills, string(cp)+" "+glyph)
	}
	return a.styles.Dim.Render("(" + strings.Join(pills, " · ") + ")")
}

func tabBorderWithBottom(left, middle, right string) lipgloss.Border {
	b := lipgloss.RoundedBorder()
	b.BottomLeft = left
	b.Bottom = middle
	b.BottomRight = right
	return b
}

var (
	tabInactiveBorder = tabBorderWithBottom("┴", "─", "┴")
	tabActiveBorder   = tabBorderWithBottom("┘", " ", "└")
)

func (a *App) tabBarView() string {
	tabs := []struct {
		label string
		t     tab
	}{
		{"inventory [1]", tabInventory},
		{"sessions [2]", tabSessions},
	}

	rendered := make([]string, len(tabs))
	for i, tb := range tabs {
		isActive := a.tab == tb.t
		style := a.styles.TabInactive
		if isActive {
			style = a.styles.TabActive
		}
		rendered[i] = style.Render(tb.label)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
}

func (a *App) statusLeftView() string {
	return a.keys.StatusHint(a.tab)
}

func (a *App) unreadNotifView() string {
	n := a.sidebar.notifs.unreadCount()
	if n < 1 {
		return ""
	}
	label := "Notifications"
	if n == 1 {
		label = "Notification"
	}
	return a.styles.Accent.Render(itoa(n) + " " + label)
}

func (a *App) statusRightView(maxW int) string {
	ids := a.svc.GetCloudProvidersIDs()
	busy := false
	for _, cp := range ids {
		if a.svc.GetCloudProviderStatus(cp) == core.CloudProviderStatusAuthenticating {
			busy = true
			break
		}
	}

	sync := a.styles.Dim.Render("✓ synced")
	if busy {
		sync = a.styles.Dim.Render("syncing…")
	}
	badge := a.unreadNotifView()
	pills := a.providerPillsView()
	user := a.loggedUserLabel

	full := joinStatusParts(a.styles, badge, sync, pills, user)
	if maxW < 1 {
		return joinStatusParts(a.styles, badge, sync)
	}
	if lipgloss.Width(full) <= maxW {
		return full
	}
	withoutUser := joinStatusParts(a.styles, badge, sync, pills)
	if lipgloss.Width(withoutUser) <= maxW {
		return withoutUser
	}
	return joinStatusParts(a.styles, badge, sync)
}

func joinStatusParts(s Styles, parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, s.Dim.Render(" · "))
}

func (a *App) bottomChromeView() string {
	w := a.layoutWidth()
	left := a.styles.Dim.Render(a.statusLeftView())
	row := clipLine(left, w)
	if a.overlay != overlaySidebar {
		avail := max(0, w-lipgloss.Width(left))
		row = joinClipRow(left, a.statusRightView(avail), w)
	}
	return boxNoWrap(a.styles.StatusBar.UnsetForeground(), row, w, 1)
}

// joinClipRow packs left and right into one w-wide line. Overflow clips the
// left side first so the right-hand status stays visible.
func joinClipRow(left, right string, w int) string {
	if w < 1 {
		return ""
	}
	lw := lipgloss.Width(left)
	rw := lipgloss.Width(right)
	if lw+rw <= w {
		return left + strings.Repeat(" ", w-lw-rw) + right
	}
	if rw >= w {
		return clipLine(right, w)
	}
	return clipLine(left, w-rw) + right
}
