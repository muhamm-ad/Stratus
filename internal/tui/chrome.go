package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/muhamm-ad/stratus/internal/core"
)

func (a *App) propagateSize() {
	h := a.contentHeight()
	a.inv.SetSize(a.width, h)
	a.audit.SetSize(a.width, h)
	a.sess.SetSize(a.width, h)
}

func (a *App) contentHeight() int {
	h := a.height - lipgloss.Height(a.topChromeView()) - lipgloss.Height(a.bottomChromeView())
	if h < 1 {
		return 1
	}
	return h
}

func (a *App) topChromeView() string {
	tabBar := a.tabBarView()

	gapWidth := max(0, a.width-lipgloss.Width(tabBar))
	identity := lipgloss.NewStyle().Inline(true).MaxWidth(gapWidth).Render(a.loggedUserLabel)
	gap := a.styles.TabInactive.
		BorderTop(false).
		BorderLeft(false).
		BorderRight(false).
		Padding(0, 0).
		Width(gapWidth).
		Align(lipgloss.Right).
		Render(lipgloss.JoinVertical(
			lipgloss.Right,
			identity,
			a.providerPillsView(),
		))
	return a.styles.Header.Width(a.width).Render(
		lipgloss.JoinHorizontal(lipgloss.Bottom, tabBar, gap),
	)
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
		{"audit [3]", tabAudit},
		{"settings [4]", tabSettings},
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
	return a.styles.FilterLine.Width(a.width).Render(line + hint)
}

func (a *App) statusLeftView() string {
	return a.styles.Dim.Render(a.keys.StatusHint(a.tab))
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
	result := fmt.Sprintf("%d/%d vms · %d sess · %s · thm:%s · %d×%d",
		len(a.inv.filteredVM), len(a.inv.allVM), len(a.sess.sessions), sync, Themes[a.themeIdx].Name, a.width, a.height)

	return a.styles.Dim.Render(result)
}

func (a *App) bottomChromeView() string {
	left := a.statusLeftView()
	right := a.statusRightView()
	gapWidth := max(0, a.width-lipgloss.Width(left))
	rightAligned := lipgloss.NewStyle().Width(gapWidth).Align(lipgloss.Right).Render(right)
	row := lipgloss.JoinHorizontal(lipgloss.Bottom, left, rightAligned)
	return a.styles.StatusBar.Width(a.width).Height(1).MaxHeight(1).Render(row)
}
