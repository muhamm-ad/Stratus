package tui

import (
	"errors"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/muhamm-ad/stratus/core"
	"github.com/muhamm-ad/stratus/service"
)

type screen int

const (
	screenLogin screen = iota
	screenApp
)

type tab int

const (
	tabInventory tab = iota
	tabSessions
	tabAudit
	tabSettings
)

type overlay int

const (
	overlayNone overlay = iota
	overlayPalette
	overlayHelp
	overlayConfirmQuit
)

// App is the root Bubble Tea model. It owns the chrome and delegates to per-tab
// sub-models. Only App implements the full tea.Model (its View returns tea.View);
// sub-models return plain strings, as recommended for children in Bubble Tea v2.
type App struct {
	gw   service.Gateway
	send func(tea.Msg) // program.Send, injected after NewProgram

	width, height int
	themeIdx      int
	styles        Styles
	keys          KeyMap

	screen  screen
	tab     tab
	overlay overlay

	login    loginModel
	inv      inventoryModel
	sess     sessionsModel
	audit    auditModel
	settings settingsModel
	palette  paletteModel
	help     helpModel
	logs     logPane
	showLogs bool

	flash     string
	flashKind string // ok|warn|err
	identity  service.Identity
	banners   []string // error banners (e.g. gcp session expired)

	searchMode bool
	searchBuf  string

	lastG time.Time // for multi-key "gg"
}

func New(gw service.Gateway) *App {
	th := Themes[0]
	a := &App{
		gw:     gw,
		keys:   DefaultKeys(),
		themeIdx: 0,
		styles: NewStyles(th),
		screen: screenLogin,
	}
	a.login = newLoginModel(gw)
	a.inv = newInventoryModel(gw)
	a.sess = newSessionsModel(gw)
	a.audit = newAuditModel(gw)
	a.settings = newSettingsModel(gw)
	a.palette = newPaletteModel()
	a.help = newHelpModel()
	a.logs = newLogPane()
	return a
}

// SetSend wires program.Send so async callbacks (device code) can inject msgs.
func (a *App) SetSend(f func(tea.Msg)) { a.send = f }

func (a *App) Init() tea.Cmd {
	// The braille spinner starts ticking immediately for the login/sync UI.
	return tea.Batch(a.login.spinner.Tick)
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		a.propagateSize()
		return a, nil

	case tea.KeyPressMsg:
		// Overlays capture keys first.
		if a.overlay != overlayNone {
			return a.updateOverlay(msg)
		}
		if a.screen == screenLogin {
			m, cmd := a.login.Update(msg, a.send)
			a.login = m
			return a, cmd
		}
		return a.updateAppKeys(msg)

	case deviceCodeMsg:
		a.login.deviceCode = core.DeviceCode(msg)
		return a, nil

	case loginResultMsg:
		if msg.err != nil {
			a.login.err = msg.err
			return a, nil
		}
		a.identity = msg.id
		a.screen = screenApp
		a.flash, a.flashKind = "welcome, "+msg.id.User+" — signed in via "+msg.id.IdP, "ok"
		// Kick off staggered per-provider sync (aws→gcp→azure).
		cmds = append(cmds,
			flashClearCmd(),
			syncProviderCmd(a.gw, "aws", 700*time.Millisecond),
			syncProviderCmd(a.gw, "gcp", 1100*time.Millisecond),
			syncProviderCmd(a.gw, "azure", 1400*time.Millisecond),
			autoRefreshCmd(),
		)
		return a, tea.Batch(cmds...)

	case vmsLoadedMsg:
		a.inv.mergeProvider(msg.provider, msg.vms)
		a.logs.add("INFO", msg.provider+" synced ("+itoa(len(msg.vms))+" VMs)")
		return a, nil

	case vmsLoadErrMsg:
		if errors.Is(msg.err, core.ErrNotAuthenticated) || errors.Is(msg.err, core.ErrExchange) {
			a.banners = append(a.banners, "▲ "+msg.provider+": session expired, VMs not loaded — press R to reconnect")
			a.logs.add("WARN", msg.provider+" token expired")
		} else {
			a.logs.add("WARN", msg.provider+": "+msg.err.Error())
		}
		return a, nil

	case connectErrMsg:
		a.flash, a.flashKind = "connect failed: "+msg.vm, "err"
		return a, flashClearCmd()

	case sessionOpenedMsg:
		return a, execSessionCmd(msg.spec)

	case sessionClosedMsg:
		a.gw.CloseSession(nil, msg.id)
		a.logs.add("INFO", "session closed: "+msg.id)
		a.flash, a.flashKind = "session closed", "ok"
		return a, flashClearCmd()

	case flashClearMsg:
		a.flash = ""
		return a, nil

	case autoRefreshMsg:
		if a.settings.autoRefresh {
			cmds = append(cmds,
				syncProviderCmd(a.gw, "aws", 0),
				syncProviderCmd(a.gw, "gcp", 0),
				syncProviderCmd(a.gw, "azure", 0),
				autoRefreshCmd(),
			)
		} else {
			cmds = append(cmds, autoRefreshCmd())
		}
		return a, tea.Batch(cmds...)

	case tokenExpiredMsg:
		a.banners = append(a.banners, "▲ "+msg.provider+": session expired, VMs not loaded — press R to reconnect")
		a.logs.add("WARN", msg.provider+" token expired")
		return a, nil
	}

	// Delegate ticks (spinner) and component msgs to the active area.
	return a.delegate(msg)
}

func (a *App) View() tea.View {
	var body string
	if a.screen == screenLogin {
		body = a.login.View(a.styles, a.width, a.height)
	} else {
		body = a.appView()
	}

	// Overlays via the Lip Gloss v2 compositor (no manual z-index).
	if a.overlay != overlayNone {
		body = a.composeOverlay(body)
	}

	v := tea.NewView(body)
	v.AltScreen = true
	v.BackgroundColor = a.styles.th.Bg
	v.WindowTitle = "stratus — multi-cloud vm gateway"
	return v
}

// appView assembles header + filter line + banners + active tab + status bar
// (+ optional log pane), using lipgloss.JoinVertical.
func (a *App) appView() string {
	header := a.headerView()
	var mid string
	switch a.tab {
	case tabInventory:
		mid = a.inv.View(a.styles, a.width, a.contentHeight())
	case tabSessions:
		mid = a.sess.View(a.styles, a.width, a.contentHeight())
	case tabAudit:
		mid = a.audit.View(a.styles, a.width, a.contentHeight())
	case tabSettings:
		mid = a.settings.View(a.styles, a.width, a.contentHeight(), a.themeIdx)
	}
	parts := []string{header}
	if a.tab != tabSettings {
		parts = append(parts, a.filterLineView())
	}
	for _, b := range a.banners {
		parts = append(parts, a.styles.ErrorBanner.Width(a.width).Render(b))
	}
	parts = append(parts, mid)
	if a.showLogs {
		parts = append(parts, a.logs.View(a.styles, a.width))
	}
	parts = append(parts, a.statusBarView())
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}
