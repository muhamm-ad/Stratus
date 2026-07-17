package tui

import (
	// "fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

func (a *App) propagateSize() {
	a.inv.tbl.SetWidth(a.width)
	a.inv.tbl.SetHeight(a.contentHeight())
}

func (a *App) contentHeight() int {
	h := a.height - 4 // header + filter + status
	if a.tab != tabSettings {
		h-- // filter line
	}
	for range a.banners {
		h--
	}
	if a.showLogs {
		h -= 4
	}
	if h < 1 {
		return 1
	}
	return h
}

func (a *App) headerView() string {
	title := a.styles.Title.Render("S T R A T U S")
	tabs := []string{
		a.tabLabel("1 inventory", tabInventory),
		a.tabLabel("2 sessions", tabSessions),
		a.tabLabel("3 audit", tabAudit),
		a.tabLabel("4 settings", tabSettings),
	}
	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	_right := ""
	if a.flash != "" {
		var flashStyle lipgloss.Style
		switch a.flashKind {
		case "ok":
			flashStyle = a.styles.OK
		case "warn":
			flashStyle = a.styles.Warn
		case "err":
			flashStyle = a.styles.Err
		default:
			flashStyle = a.styles.Accent
		}
		_right = a.styles.FilterLine.Width(a.width).Render(flashStyle.Render(a.flash))
	}

	right := lipgloss.NewStyle().Width(max(0, a.width-lipgloss.Width(title)-lipgloss.Width(tabBar)-5)).Align(lipgloss.Right).Render(_right)
	return a.styles.Header.Width(a.width).Render(
		lipgloss.JoinHorizontal(lipgloss.Center, title, "  ", tabBar, " ", right),
	)
}

func (a *App) tabLabel(label string, t tab) string {
	if a.tab == t {
		return a.styles.TabActive.Render(label)
	}
	return a.styles.TabInactive.Render(label)
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
	hint := a.styles.Dim.Render(" · x clear")
	return a.styles.FilterLine.Width(a.width).Render(line + hint)
}

func (a *App) statusBarView() string {
	var hint string
	switch a.tab {
	case tabInventory:
		hint = HintInventory
	case tabSessions:
		hint = HintSessions
	case tabAudit:
		hint = HintAudit
	case tabSettings:
		hint = HintSettings
	}
	theme := a.styles.Dim.Render("theme: " + Themes[a.themeIdx].Name)
	left := a.styles.Dim.Render(hint)
	right := theme
	gap := max(0, a.width-lipgloss.Width(left)-lipgloss.Width(right))
	return a.styles.StatusBar.Width(a.width).Render(left + strings.Repeat(" ", gap) + right)
}

// func (a *App) inventoryFilterLine() string {
// 	return a.filterLineView()
// }

// func max(a, b int) int {
// 	if a > b {
// 		return a
// 	}
// 	return b
// }

// func formatCount(n int, noun string) string {
// 	return fmt.Sprintf("%d %s", n, noun)
// }
