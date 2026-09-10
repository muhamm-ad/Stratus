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
	if a.searchMode {
		switch a.keys.Nav.Match(msg) {
		case ActionSelect:
			a.searchMode = false
			a.inv.query = a.searchBuf
			a.inv.recompute()
			return a, a.log("INFO", "search: "+a.inv.query)
		case ActionBack:
			a.searchMode = false
			return a, a.log("INFO", "search cancelled")
		}
		if msg.String() == "backspace" {
			if len(a.searchBuf) > 0 {
				a.searchBuf = a.searchBuf[:len(a.searchBuf)-1]
			}
			return a, nil
		}
		if keyStr := msg.String(); len(keyStr) == 1 {
			a.searchBuf += keyStr
		}
		return a, nil
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
		if a.keys.Inventory.Match(msg) == ActionSearch {
			a.searchMode = true
			a.searchBuf = ""
			return a, a.log("INFO", "search opened")
		}
		prevProv, prevState, prevRegion := a.inv.fProvider, a.inv.fState, a.inv.fRegion
		prevSort, prevAsc, prevDetail := a.inv.sortK, a.inv.sortAsc, a.inv.detailOn
		m, cmd, intent := a.inv.Update(msg, a.keys, a.styles)
		a.inv = m
		var extra tea.Cmd
		switch {
		case a.inv.fProvider != prevProv:
			val := string(a.inv.fProvider)
			if val == "" {
				val = "all"
			}
			extra = a.log("INFO", "filter provider: "+val)
		case a.inv.fState != prevState:
			val := string(a.inv.fState)
			if val == "" {
				val = "all"
			}
			extra = a.log("INFO", "filter state: "+val)
		case a.inv.fRegion != prevRegion:
			val := string(a.inv.fRegion)
			if val == "" {
				val = "all"
			}
			extra = a.log("INFO", "filter region: "+val)
		case a.inv.sortK != prevSort || a.inv.sortAsc != prevAsc:
			dir := "desc"
			if a.inv.sortAsc {
				dir = "asc"
			}
			names := [...]string{"none", "name", "provider", "region", "type", "state"}
			name := "none"
			if int(a.inv.sortK) < len(names) {
				name = names[int(a.inv.sortK)]
			}
			extra = a.log("INFO", "sort: "+name+" "+dir)
		case a.inv.detailOn != prevDetail:
			if a.inv.detailOn {
				extra = a.log("INFO", "vm detail opened")
			} else {
				extra = a.log("INFO", "vm detail closed")
			}
		}
		return a.handleIntent(intent, tea.Batch(cmd, extra))
	case tabSessions:
		return a, a.sess.Update(msg, a.keys)
	case tabAudit:
		return a, a.audit.Update(msg)
	case tabSettings:
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
			return a, nil
		}
		return a, tea.Batch(cmds...)
	}
	return a, nil
}

func (a *App) handleGlobal(act Action) (tea.Model, tea.Cmd) {
	switch act {
	case ActionTabInventory:
		a.tab = tabInventory
		return a, a.log("INFO", "tab: "+tabName(a.tab))
	case ActionTabSessions:
		a.tab = tabSessions
		return a, a.log("INFO", "tab: "+tabName(a.tab))
	case ActionTabAudit:
		a.tab = tabAudit
		return a, a.log("INFO", "tab: "+tabName(a.tab))
	case ActionTabSettings:
		a.tab = tabSettings
		return a, a.log("INFO", "tab: "+tabName(a.tab))
	case ActionPalette:
		a.overlay = overlayPalette
		return a, tea.Batch(a.cmdPalette.open(), a.log("INFO", "command palette"))
	case ActionHelp:
		a.overlay = overlayHelp
		a.help.scroll = 0
		return a, a.log("INFO", "help")
	case ActionLogs:
		return a, a.toggleLogAlerts()
	case ActionSidebar:
		a.overlay = overlaySidebar
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

func (a *App) jumpTop() {
	switch a.tab {
	case tabInventory:
		a.inv.tbl.GotoTop()
		a.inv.syncTableRows()
	case tabAudit:
		a.audit.tbl.GotoTop()
		a.audit.syncTableRows()
	case tabSessions:
		if n := len(a.sess.list.Items()); n > 0 {
			a.sess.list.Select(0)
		}
	case tabSettings:
		a.settings.cursor = 0
	}
}

func (a *App) jumpBottom() {
	switch a.tab {
	case tabInventory:
		a.inv.tbl.GotoBottom()
		a.inv.syncTableRows()
	case tabAudit:
		a.audit.tbl.GotoBottom()
		a.audit.syncTableRows()
	case tabSessions:
		if n := len(a.sess.list.Items()); n > 0 {
			a.sess.list.Select(n - 1)
		}
	case tabSettings:
		a.settings.cursor = a.settings.maxCursor()
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
		case ActionSidebarLogs:
			a.sidebar.setTab(sidebarTabLogs)
		}
		switch a.keys.Nav.Match(msg) {
		case ActionMoveUp:
			a.sidebar.scrollBy(-1, a.styles, a.height, a.logs)
		case ActionMoveDown:
			a.sidebar.scrollBy(1, a.styles, a.height, a.logs)
		case ActionPageUp:
			a.sidebar.scrollBy(-a.sidebar.pageSize(a.styles, a.height), a.styles, a.height, a.logs)
		case ActionPageDown:
			a.sidebar.scrollBy(a.sidebar.pageSize(a.styles, a.height), a.styles, a.height, a.logs)
		case ActionJumpTop:
			a.sidebar.scroll = 0
		case ActionJumpBottom:
			a.sidebar.scroll = a.sidebar.maxScroll(a.styles, a.height, a.logs)
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
	case "audit":
		a.tab = tabAudit
	case "settings":
		a.tab = tabSettings
	case "aws":
		a.inv.fProvider = "aws"
		a.inv.recompute()
	case "azure":
		a.inv.fProvider = "azure"
		a.inv.recompute()
	case "gcp":
		a.inv.fProvider = "gcp"
		a.inv.recompute()
	case "all":
		a.inv.fProvider = ""
		a.inv.recompute()
	case "running":
		a.inv.fState = "running"
		a.inv.recompute()
	case "stopped":
		a.inv.fState = "stopped"
		a.inv.recompute()
	case "clear":
		a.inv.clearFilters()
		a.inv.recompute()
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
	case "help":
		a.overlay = overlayHelp
		a.help.scroll = 0
	case "theme":
		a.setTheme((a.themeIdx + 1) % len(Themes))
	case "quit":
		a.overlay = overlayConfirmQuit
		a.quitFocus = quitBtnQuit
	case "reconnect":
		return a.reconnectExpiredCmd()
	case "region":
		a.inv.cycleRegion()
		a.inv.recompute()
	case "tag":
		return a.log("WARN", "tag filter not yet implemented")
	}
	return nil
}
