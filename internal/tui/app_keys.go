package tui

import (
	"context"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/muhamm-ad/stratus/internal/core"
	"go.dalton.dog/bubbleup/v2"
)

// reconnectProviderCmd re-authenticates a single provider via the service
// layer, reporting success/failure as a providerReconnect{OK,Err}Msg.
// Status transitions (Authenticating → Authenticated/Error) are owned by
// the provider itself inside Authenticate.
func (a *App) reconnectProviderCmd(cp core.CloudProviderID) tea.Cmd {
	return func() tea.Msg {
		if err := a.svc.ReconnectProvider(context.Background(), cp); err != nil {
			return providerReconnectErrMsg{provider: cp, err: err}
		}
		return providerReconnectOKMsg{provider: cp}
	}
}

// reconnectExpiredCmd reconnects the first provider currently in
// CloudProviderStatusError (mirrors the mockup's "find the first errored
// provider" behavior for the R key / :reconnect palette command).
func (a *App) reconnectExpiredCmd() tea.Cmd {
	for _, cp := range a.svc.GetCloudProvidersIDs() {
		if a.svc.GetCloudProviderStatus(cp) == core.CloudProviderStatusError {
			return tea.Batch(a.log("INFO", "reconnecting "+string(cp)), a.reconnectProviderCmd(cp))
		}
	}
	return tea.Batch(
		a.log("WARN", "no expired provider to reconnect"),
		a.notify(bubbleup.WarnKey, "no expired provider to reconnect"),
	)
}

func (a *App) toggleLogAlerts() tea.Cmd {
	a.settings.logAlerts = !a.settings.logAlerts
	return a.logLogAlertsState()
}

func (a *App) logLogAlertsState() tea.Cmd {
	state := "off"
	if a.settings.logAlerts {
		state = "on"
	}
	msg := "log alerts " + state
	if a.settings.logAlerts {
		return a.log("INFO", msg)
	}
	a.logs.add("INFO", msg)
	return a.notify(bubbleup.InfoKey, msg)
}

func (a *App) updateAppKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if a.tab == tabInventory && a.inv.QueryFocused() {
		return a.updateInventoryKeys(msg)
	}

	// Inventory steals "?" for filter help; ctrl+h still opens app help.
	if a.tab == tabInventory && a.keys.Inventory.Match(msg) == ActionQueryHelp {
		return a, a.openQueryHelp()
	}

	// "gg" is a two-stroke gesture; a lone "g" must not jump yet.
	if msg.String() == "g" {
		if time.Since(a.lastG) < 450*time.Millisecond {
			a.jumpTop()
			a.lastG = time.Time{}
			return a, nil
		}
		a.lastG = time.Now()
		return a, ggResetCmd()
	}

	if act := a.keys.Global.Match(msg); act != ActionNone {
		return a.handleGlobal(act)
	}

	switch a.keys.Nav.Match(msg) {
	case ActionJumpTop:
		a.jumpTop()
		return a, nil
	case ActionJumpBottom:
		a.jumpBottom()
		return a, nil
	}

	switch a.tab {
	case tabInventory:
		return a.updateInventoryKeys(msg)
	case tabSessions:
		return a, a.sess.Update(msg, a.keys)
	}
	return a, nil
}

func (a *App) updateInventoryKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	prevQ := a.inv.QueryString()
	prevFocus := a.inv.QueryFocused()
	prevDetail := a.inv.detailOn
	m, cmd, intent := a.inv.Update(msg, a.keys, a.styles)
	a.inv = m
	var extra tea.Cmd
	switch {
	case !prevFocus && a.inv.QueryFocused():
		extra = a.log("INFO", "query opened")
	case prevFocus && !a.inv.QueryFocused():
		q := a.inv.QueryString()
		if q == "" {
			extra = a.log("INFO", "query closed")
		} else {
			extra = a.log("INFO", "query: "+q)
		}
	case !a.inv.QueryFocused() && a.inv.QueryString() != prevQ:
		q := a.inv.QueryString()
		if q == "" {
			extra = a.log("INFO", "query cleared")
		} else {
			extra = a.log("INFO", "query: "+q)
		}
	case a.inv.detailOn != prevDetail:
		if a.inv.detailOn {
			extra = a.log("INFO", "vm detail opened")
		} else {
			extra = a.log("INFO", "vm detail closed")
		}
	}
	return a.handleIntent(intent, tea.Batch(cmd, extra))
}

func (a *App) openHelp() tea.Cmd {
	a.overlay = overlayHelp
	a.help.query = false
	a.help.scroll = 0
	return a.log("INFO", "help")
}

func (a *App) openQueryHelp() tea.Cmd {
	a.overlay = overlayHelp
	a.help.query = true
	a.help.scroll = 0
	return a.log("INFO", "query help")
}

func (a *App) handleGlobal(act Action) (tea.Model, tea.Cmd) {
	switch act {
	case ActionTabInventory:
		a.tab = tabInventory
		return a, a.log("INFO", "tab: "+tabName(a.tab))
	case ActionTabSessions:
		a.tab = tabSessions
		return a, a.log("INFO", "tab: "+tabName(a.tab))
	case ActionPalette:
		a.overlay = overlayPalette
		return a, tea.Batch(a.cmdPalette.open(), a.log("INFO", "command palette"))
	case ActionHelp:
		return a, a.openHelp()
	case ActionLogs:
		return a, a.toggleLogAlerts()
	case ActionSidebar:
		a.overlay = overlaySidebar
		a.seeNotifsIfVisible()
		return a, a.log("INFO", "sidebar")
	case ActionTheme:
		a.setTheme((a.themeIdx + 1) % len(Themes))
		return a, a.log("INFO", "theme: "+Themes[a.themeIdx].Name)
	case ActionQuit:
		a.overlay = overlayConfirmQuit
		a.quitFocus = quitBtnQuit
		return a, a.log("INFO", "sign-out confirm")
	case ActionReconnect:
		return a, a.reconnectExpiredCmd()
	case ActionForceQuit:
		return a, tea.Quit
	}
	return a, nil
}

func (a *App) updateSidebarSettings(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch a.keys.Nav.Match(msg) {
	case ActionPageUp:
		a.settings.cursor = max(0, a.settings.cursor-a.sidebar.pageSize(a.styles, a.height))
		a.revealSettingsCursor()
		return a, nil
	case ActionPageDown:
		a.settings.cursor = min(a.settings.maxCursor(), a.settings.cursor+a.sidebar.pageSize(a.styles, a.height))
		a.revealSettingsCursor()
		return a, nil
	}
	cmd := a.applySettingsMsg(msg)
	a.revealSettingsCursor()
	return a, cmd
}

func (a *App) applySettingsMsg(msg tea.KeyPressMsg) tea.Cmd {
	prevRefresh := a.settings.autoRefresh
	prevAlerts := a.settings.logAlerts
	sm, scmd, intent := a.settings.Update(msg, a.themeIdx, a.styles, a.keys)
	a.settings = sm
	var cmds []tea.Cmd
	if a.settings.autoRefresh != prevRefresh {
		state := "off"
		if a.settings.autoRefresh {
			state = "on"
		}
		cmds = append(cmds, a.log("INFO", "auto-refresh "+state))
	}
	if a.settings.logAlerts != prevAlerts {
		cmds = append(cmds, a.logLogAlertsState())
	}
	if scmd != nil {
		a.setTheme(scmd.themeIdx)
		cmds = append(cmds, a.log("INFO", "theme: "+Themes[a.themeIdx].Name))
	}
	_, intentCmd := a.handleIntent(intent, nil)
	if intentCmd != nil {
		cmds = append(cmds, intentCmd)
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

func (a *App) revealSettingsCursor() {
	listH := a.sidebar.listHeight(a.styles, a.height)
	c := a.settings.cursor
	if c < a.sidebar.scroll {
		a.sidebar.scroll = c
	} else if c >= a.sidebar.scroll+listH {
		a.sidebar.scroll = c - listH + 1
	}
	maxScroll := a.sidebar.maxScroll(a.styles, a.height, a.logs, a.settings, a.themeIdx)
	if a.sidebar.scroll > maxScroll {
		a.sidebar.scroll = maxScroll
	}
	if a.sidebar.scroll < 0 {
		a.sidebar.scroll = 0
	}
}

func (a *App) jumpTop() {
	switch a.tab {
	case tabInventory:
		a.inv.tbl.GotoTop()
		a.inv.syncTableRows()
	case tabSessions:
		if n := len(a.sess.list.Items()); n > 0 {
			a.sess.list.Select(0)
		}
	}
}

func (a *App) jumpBottom() {
	switch a.tab {
	case tabInventory:
		a.inv.tbl.GotoBottom()
		a.inv.syncTableRows()
	case tabSessions:
		if n := len(a.sess.list.Items()); n > 0 {
			a.sess.list.Select(n - 1)
		}
	}
}

func (a *App) handleIntent(intent appIntent, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	switch intent.kind {
	case intentConnect:
		cmds = append(cmds, a.log("INFO", fmt.Sprintf("connect %d vm(s)", len(intent.targets))))
		cmds = append(cmds, a.connectCmd(intent.targets)...)
	case intentStop:
		cmds = append(cmds, a.log("INFO", fmt.Sprintf("stop %d vm(s)", len(intent.targets))))
		cmds = append(cmds, a.stopCmd(intent.targets)...)
	case intentStart:
		cmds = append(cmds, a.log("INFO", fmt.Sprintf("start %d vm(s)", len(intent.targets))))
		cmds = append(cmds, a.startCmd(intent.targets)...)
	case intentRefresh:
		cmds = append(cmds, a.log("INFO", "refresh inventory"))
		cmds = append(cmds, syncProviderCmds(a.svc, false)...)
	case intentReconnect:
		cmds = append(cmds, a.log("INFO", "reconnect "+intent.provider))
		cmds = append(cmds, a.reconnectProviderCmd(core.CloudProviderID(intent.provider)))
	case intentQueryHelp:
		cmds = append(cmds, a.openQueryHelp())
	}
	if len(cmds) == 0 {
		return a, nil
	}
	return a, tea.Batch(cmds...)
}

func (a *App) connectCmd(targets []core.VM) []tea.Cmd { // FIXME: Implement this
	// svc := a.svc
	var cmds []tea.Cmd
	// var opened, skipped int
	// for _, vm := range targets {
	// 	if !vm.CanConnect {
	// 		skipped++
	// 		continue
	// 	}
	// 	opened++
	// 	vm := vm
	// 	cmds = append(cmds, func() tea.Msg {
	// 		spec, err := gw.OpenSession(context.Background(), vm.ID)
	// 		if err != nil {
	// 			return connectErrMsg{vm: vm.Name, err: err}
	// 		}
	// 		return sessionOpenedMsg{spec: spec}
	// 	})
	// }
	// if opened > 0 || skipped > 0 {
	// 	msg := fmt.Sprintf("%d opened", opened)
	// 	if skipped > 0 {
	// 		msg += fmt.Sprintf(" · %d skipped (no permission)", skipped)
	// 	}
	// 	cmds = append(cmds, a.notify(bubbleup.InfoKey, msg))
	// }
	return cmds
}

func (a *App) stopCmd(targets []core.VM) []tea.Cmd { // FIXME: Implement this
	_ = targets
	// for _, vm := range targets {
	// 	_ = a.gw.StopVM(context.Background(), vm.ID)
	// }
	// if len(targets) > 0 {
	// 	cmds = append(cmds, a.notify(bubbleup.WarnKey, fmt.Sprintf("stop requested for %d vm(s)", len(targets))))
	// }
	return nil
}

func (a *App) startCmd(targets []core.VM) []tea.Cmd { // FIXME: Implement this
	_ = targets
	// for _, vm := range targets {
	// 	_ = a.gw.StartVM(context.Background(), vm.ID)
	// }
	// if len(targets) > 0 {
	// 	cmds = append(cmds, a.notify(bubbleup.InfoKey, fmt.Sprintf("start requested for %d vm(s)", len(targets))))
	// }
	return nil
}

type connectErrMsg struct {
	vm  string
	err error
}

func (a *App) delegate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case ggResetMsg:
		a.lastG = time.Time{}
	}
	if a.screen == screenLogin {
		var cmd tea.Cmd
		a.login.spinner, cmd = a.login.spinner.Update(msg)
		return a, cmd
	}
	if cmd := a.inv.HandleMsg(msg); cmd != nil {
		return a, cmd
	}
	return a, nil
}

func (a *App) updateOverlay(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch a.overlay {
	case overlayPalette:
		if a.keys.Nav.Match(msg) == ActionBack {
			a.overlay = overlayNone
			return a, a.log("INFO", "overlay closed")
		}
		m, picked := a.cmdPalette.Update(msg)
		a.cmdPalette = m
		if picked != nil {
			a.overlay = overlayNone
			return a, tea.Batch(a.log("INFO", "cmd: "+picked.name), a.runPaletteCommand(*picked))
		}
	case overlayHelp:
		if a.keys.Global.Match(msg) == ActionHelp || a.keys.Nav.Match(msg) == ActionBack {
			a.overlay = overlayNone
			a.help.scroll = 0
			a.help.query = false
			return a, a.log("INFO", "overlay closed")
		}
		switch a.keys.Nav.Match(msg) {
		case ActionMoveUp:
			a.help.scrollBy(-1, a.styles, a.keys, a.width, a.height)
		case ActionMoveDown:
			a.help.scrollBy(1, a.styles, a.keys, a.width, a.height)
		case ActionPageUp:
			a.help.scrollBy(-a.helpBodyPage(), a.styles, a.keys, a.width, a.height)
		case ActionPageDown:
			a.help.scrollBy(a.helpBodyPage(), a.styles, a.keys, a.width, a.height)
		case ActionJumpTop:
			a.help.scroll = 0
		case ActionJumpBottom:
			a.help.scroll = a.help.maxScroll(a.styles, a.keys, a.width, a.height)
		}
	case overlaySidebar:
		if a.keys.Global.Match(msg) == ActionSidebar || a.keys.Nav.Match(msg) == ActionBack {
			a.overlay = overlayNone
			return a, a.log("INFO", "overlay closed")
		}
		switch a.keys.Sidebar.Match(msg) {
		case ActionSidebarNotif:
			a.sidebar.setTab(sidebarTabNotif)
			a.seeNotifsIfVisible()
		case ActionSidebarLogs:
			a.sidebar.setTab(sidebarTabLogs)
		case ActionSidebarSettings:
			a.sidebar.setTab(sidebarTabSettings)
		}
		if a.sidebar.tab == sidebarTabSettings {
			return a.updateSidebarSettings(msg)
		}
		switch a.keys.Nav.Match(msg) {
		case ActionMoveUp:
			a.sidebar.scrollBy(-1, a.styles, a.height, a.logs, a.settings, a.themeIdx)
		case ActionMoveDown:
			a.sidebar.scrollBy(1, a.styles, a.height, a.logs, a.settings, a.themeIdx)
		case ActionPageUp:
			a.sidebar.scrollBy(-a.sidebar.pageSize(a.styles, a.height), a.styles, a.height, a.logs, a.settings, a.themeIdx)
		case ActionPageDown:
			a.sidebar.scrollBy(a.sidebar.pageSize(a.styles, a.height), a.styles, a.height, a.logs, a.settings, a.themeIdx)
		case ActionJumpTop:
			a.sidebar.scroll = 0
		case ActionJumpBottom:
			a.sidebar.scroll = a.sidebar.maxScroll(a.styles, a.height, a.logs, a.settings, a.themeIdx)
		}
	case overlayConfirmQuit:
		act := a.keys.Overlay.Match(msg)
		if act == ActionNone {
			act = a.keys.Nav.Match(msg)
		}
		switch act {
		case ActionOverlayNext, ActionOverlayPrev:
			a.quitFocus = 1 - a.quitFocus
		case ActionOverlayConfirm, ActionSelect:
			if a.quitFocus == quitBtnQuit {
				return a, tea.Batch(a.log("INFO", "signing out"), tea.Quit)
			}
			a.overlay = overlayNone
			return a, a.log("INFO", "overlay closed")
		case ActionOverlayYes:
			return a, tea.Batch(a.log("INFO", "signing out"), tea.Quit)
		case ActionOverlayNo, ActionBack:
			a.overlay = overlayNone
			return a, a.log("INFO", "overlay closed")
		}
	}
	return a, nil
}

func (a *App) helpBodyPage() int {
	return helpBodyHeight(a.styles, a.height)
}

func (a *App) runPaletteCommand(c command) tea.Cmd {
	switch c.name {
	case "inventory":
		a.tab = tabInventory
	case "sessions":
		a.tab = tabSessions
	case "settings":
		a.overlay = overlaySidebar
		a.sidebar.setTab(sidebarTabSettings)
	case "aws":
		a.inv.applyProvider("aws")
	case "azure":
		a.inv.applyProvider("azure")
	case "gcp":
		a.inv.applyProvider("gcp")
	case "all":
		a.inv.applyProvider("")
	case "running":
		a.inv.applyState("running")
	case "stopped":
		a.inv.applyState("stopped")
	case "clear":
		a.inv.clearFilters()
	case "connect":
		_, cmd := a.handleIntent(appIntent{kind: intentConnect, targets: a.inv.selectedVMs()}, nil)
		return cmd
	case "start":
		_, cmd := a.handleIntent(appIntent{kind: intentStart, targets: a.inv.selectedVMs()}, nil)
		return cmd
	case "stop":
		_, cmd := a.handleIntent(appIntent{kind: intentStop, targets: a.inv.selectedVMs()}, nil)
		return cmd
	case "refresh":
		_, cmd := a.handleIntent(appIntent{kind: intentRefresh}, nil)
		return cmd
	case "logs":
		return a.toggleLogAlerts()
	case "sidebar":
		a.overlay = overlaySidebar
		a.seeNotifsIfVisible()
	case "help":
		return a.openHelp()
	case "query-help":
		return a.openQueryHelp()
	case "theme":
		a.setTheme((a.themeIdx + 1) % len(Themes))
	case "quit":
		a.overlay = overlayConfirmQuit
		a.quitFocus = quitBtnQuit
	case "reconnect":
		return a.reconnectExpiredCmd()
	case "region":
		a.inv.cycleRegion()
	case "tag":
		return a.log("INFO", "tag filter: type tag=key:value in the query bar")
	}
	return nil
}
