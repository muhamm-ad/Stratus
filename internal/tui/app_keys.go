package tui

import (
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/muhamm-ad/stratus/internal/core"
)

func (a *App) updateAppKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	keyStr := msg.String()

	if a.searchMode {
		switch keyStr {
		case "esc":
			a.searchMode = false
			return a, nil
		case "enter":
			a.searchMode = false
			a.inv.query = a.searchBuf
			a.inv.recompute()
			return a, nil
		case "backspace":
			if len(a.searchBuf) > 0 {
				a.searchBuf = a.searchBuf[:len(a.searchBuf)-1]
			}
			return a, nil
		default:
			if len(keyStr) == 1 {
				a.searchBuf += keyStr
			}
			return a, nil
		}
	}

	if keyStr == "g" {
		if time.Since(a.lastG) < 450*time.Millisecond {
			a.inv.tbl.GotoTop()
			a.lastG = time.Time{}
			return a, nil
		}
		a.lastG = time.Now()
		return a, ggResetCmd()
	}

	switch {
	case key.Matches(msg, a.keys.Tab1):
		a.tab = tabInventory
		return a, nil
	case key.Matches(msg, a.keys.Tab2):
		a.tab = tabSessions
		return a, nil
	case key.Matches(msg, a.keys.Tab3):
		a.tab = tabAudit
		return a, nil
	case key.Matches(msg, a.keys.Tab4):
		a.tab = tabSettings
		return a, nil
	case key.Matches(msg, a.keys.Search):
		if a.tab == tabInventory {
			a.searchMode = true
			a.searchBuf = ""
		}
		return a, nil
	case key.Matches(msg, a.keys.Cmd):
		a.overlay = overlayPalette
		return a, a.palette.open()
	case key.Matches(msg, a.keys.Help):
		a.overlay = overlayHelp
		return a, nil
	case key.Matches(msg, a.keys.Logs):
		a.showLogs = !a.showLogs
		return a, nil
	case key.Matches(msg, a.keys.Theme):
		a.setTheme((a.themeIdx + 1) % len(Themes))
		return a, nil
	case key.Matches(msg, a.keys.Quit):
		a.overlay = overlayConfirmQuit
		return a, nil
	case key.Matches(msg, a.keys.Reconnect):
		return a, tea.Batch(
			syncProviderCmds(a.svc, false)...,
		)
	}

	switch a.tab {
	case tabInventory:
		m, cmd, intent := a.inv.Update(msg, a.keys, a.styles)
		a.inv = m
		return a.handleIntent(intent, cmd)
	case tabSessions:
		return a, a.sess.Update(msg, a.keys)
	case tabAudit:
		return a, a.audit.Update(msg)
	case tabSettings:
		sm, scmd := a.settings.Update(msg, a.themeIdx, a.styles)
		a.settings = sm
		if scmd != nil {
			a.setTheme(scmd.themeIdx)
		}
		return a, nil
	}
	return a, nil
}

func (a *App) handleIntent(intent appIntent, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	switch intent.kind {
	case intentConnect:
		cmds = append(cmds, a.connectCmd(intent.targets)...)
	case intentStop:
		cmds = append(cmds, a.stopCmd(intent.targets)...)
	case intentRefresh:
		cmds = append(cmds,
			syncProviderCmds(a.svc, false)...,
		)
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
	// 	a.flash, a.flashKind = msg, "ok"
	// 	cmds = append(cmds, flashClearCmd())
	// }
	return cmds
}

func (a *App) stopCmd(targets []core.VM) []tea.Cmd { // FIXME: Implement this
	// for _, vm := range targets {
	// 	_ = a.gw.StopVM(context.Background(), vm.ID)
	// }
	// if len(targets) > 0 {
	// 	a.flash, a.flashKind = fmt.Sprintf("stop requested for %d vm(s)", len(targets)), "warn"
	// }
	return []tea.Cmd{flashClearCmd()}
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
	switch msg.String() {
	case "esc", "n":
		a.overlay = overlayNone
		return a, nil
	}
	switch a.overlay {
	case overlayPalette:
		m, picked := a.palette.Update(msg)
		a.palette = m
		if picked != nil {
			a.overlay = overlayNone
			return a, a.runPaletteCommand(*picked)
		}
	case overlayHelp:
		if msg.String() == "?" || msg.String() == "enter" {
			a.overlay = overlayNone
		}
	case overlayConfirmQuit:
		switch msg.String() {
		case "enter", "y":
			return a, tea.Quit
		}
	}
	return a, nil
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
		_, cmd := a.handleIntent(appIntent{kind: intentConnect, targets: a.inv.connectTargets()}, nil)
		return cmd
	case "refresh":
		return tea.Batch(
			syncProviderCmds(a.svc, false)...,
		)
	case "logs":
		a.showLogs = !a.showLogs
	case "help":
		a.overlay = overlayHelp
	case "theme":
		a.setTheme((a.themeIdx + 1) % len(Themes))
	case "quit":
		a.overlay = overlayConfirmQuit
	case "reconnect":
		return tea.Batch(
			syncProviderCmds(a.svc, false)...,
		)
	case "region":
		a.inv.cycleRegion()
		a.inv.recompute()
	case "tag":
		// tag filter not yet implemented
	}
	return nil
}
