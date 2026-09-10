package tui

import (
	"fmt"
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
	h := a.layoutHeight() - lipgloss.Height(a.topChromeView()) - lipgloss.Height(a.bottomChromeView())
	if h < 1 {
		return 1
	}
	return h
}

func (a *App) topChromeView() string {
	tabBar := a.tabBarView()
	w := a.layoutWidth()

	gapWidth := max(0, w-lipgloss.Width(tabBar))
	handle := ""
	if a.overlay != overlaySidebar {
		handle = sidebarHandleView(a.styles, false)
	}
	const chromeRightPad = 2
	textW := max(0, gapWidth-chromeRightPad)
	pad := strings.Repeat(" ", chromeRightPad)
	identity := lipgloss.NewStyle().Inline(true).MaxWidth(textW).Render(a.loggedUserLabel) + pad
	pills := lipgloss.NewStyle().Inline(true).MaxWidth(max(0, textW-1)).Render(a.providerPillsView()) + pad
	ruleW := max(0, gapWidth-lipgloss.Width(handle))
	rule := lipgloss.NewStyle().Foreground(a.styles.th.Border).Render(strings.Repeat("─", ruleW))
	underline := rule
	if handle != "" {
		underline = lipgloss.JoinHorizontal(lipgloss.Bottom, rule, handle)
	}
	gap := lipgloss.JoinVertical(lipgloss.Right, identity, pills, underline)
	if gapWidth == 0 {
		return boxNoWrap(a.styles.Header, tabBar, w, max(1, lipgloss.Height(tabBar)))
	}
	gap = lipgloss.NewStyle().Width(gapWidth).MaxWidth(gapWidth).MaxHeight(lipgloss.Height(gap)).Align(lipgloss.Right).Render(gap)
	row := lipgloss.JoinHorizontal(lipgloss.Bottom, tabBar, gap)
	return boxNoWrap(a.styles.Header, row, w, max(1, lipgloss.Height(row)))
}

func (a *App) providerPillsView() string {
	ids := a.svc.GetCloudProvidersIDs()
	pills := make([]string, 0, len(ids))
	for _, cp := range ids {
		var glyph string
		var glyphStyle lipgloss.Style
		switch a.svc.GetCloudProviderStatus(cp) {
		case core.CloudProviderStatusAuthenticated:
			glyph, glyphStyle = "✓", a.styles.OK
		case core.CloudProviderStatusAuthenticating:
			glyph, glyphStyle = SpinnerFrames[0], a.styles.Warn
		case core.CloudProviderStatusError:
			glyph, glyphStyle = "!", a.styles.Err
		default:
			glyph, glyphStyle = "?", a.styles.Dim
		}
		cpColored := a.styles.Dim.Foreground(ProviderColor(cp)).Render(string(cp))
		pills = append(pills, cpColored+" "+glyphStyle.Render(glyph))
	}
	return strings.Join(pills, a.styles.Dim.Render(" · "))
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

func (a *App) filterLineView() string {
	var chips []string
	provider := string(a.inv.fProvider)
	state := string(a.inv.fState)
	region := string(a.inv.fRegion)
	query := a.inv.query

	if a.tab == tabInventory {
		if a.searchMode {
			chips = append(chips, "/"+a.searchBuf+"▌")
		} else {
			if provider != "" {
				chips = append(chips, "provider: "+provider)
			}
			if state != "" && state != "all" {
				chips = append(chips, "state: "+state)
			}
			if region != "" {
				chips = append(chips, "region: "+region)
			}
			if query != "" {
				chips = append(chips, "/"+query)
			}
		}
	}
	line := "filters: all"
	if len(chips) > 0 {
		line = strings.Join(chips, " · ")
	}
	hint := a.styles.Dim.Render(" · " + keyed(a.keys.Inventory.ClearFilters, "clear"))
	w := a.layoutWidth()
	return boxNoWrap(a.styles.FilterLine, clipLine(line+hint, w), w, 1)
}

func (a *App) statusLeftView() string {
	return a.keys.StatusHint(a.tab)
}

func (a *App) statusRightView() string {
	ids := a.svc.GetCloudProvidersIDs()
	busy := false
	for _, cp := range ids {
		if a.svc.GetCloudProviderStatus(cp) == core.CloudProviderStatusAuthenticating {
			busy = true
			break
		}
	}
	sync := "✓ synced"
	if busy {
		sync = "syncing…"
	}
	// result := fmt.Sprintf("%d/%d vms · %d sess · %s · thm:%s · %d×%d",
	// 	len(a.inv.filteredVM), len(a.inv.allVM), len(a.sess.sessions), sync, Themes[a.themeIdx].Name, a.width, a.height)

	return fmt.Sprintf("%d/%d vms · %d sess · %s",
		len(a.inv.filteredVM), len(a.inv.allVM), len(a.sess.sessions), sync)
}

func (a *App) bottomChromeView() string {
	w := a.layoutWidth()
	left := a.statusLeftView()
	if a.overlay == overlaySidebar {
		return boxNoWrap(a.styles.StatusBar, clipLine(left, w), w, 1)
	}
	row := joinClipRow(left, a.statusRightView(), w)
	return boxNoWrap(a.styles.StatusBar, row, w, 1)
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
