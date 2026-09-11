package tui

import (
	"context"
	"errors"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/muhamm-ad/stratus/internal/core"
	"github.com/muhamm-ad/stratus/internal/service"
	"go.dalton.dog/bubbleup/v2"
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
)

type overlay int

const (
	overlayNone overlay = iota
	overlayPalette
	overlayHelp
	overlayConfirmQuit
	overlaySidebar
)

// App is the root Bubble Tea model. It owns the chrome and delegates to per-tab sub-models.
// Only App implements the full tea.Model (its View returns tea.View);
// sub-models return plain strings, as recommended for children in Bubble Tea v2.
type App struct {
	svc  *service.Service
	send func(tea.Msg) // program.Send, injected after NewProgram

	width, height int
	panX, panY    int // viewport into the min-size layout when the window is smaller
	themeIdx      int
	styles        Styles
	keys          KeyMap

	screen    screen
	tab       tab
	overlay   overlay
	quitFocus int // 0 = Sign out, 1 = Cancel

	login      loginModel
	inv        inventoryModel
	sess       sessionsModel
	settings   settingsModel
	cmdPalette paletteModel
	help       helpModel
	sidebar    sidebarModel
	logs       logPane

	alert    bubbleup.AlertModel
	identity core.IdentityProvider

	loggedUserLabel string

	lastG time.Time // for multi-key "gg"
}

func New(svc *service.Service) *App {
	th := Themes[3]
	a := &App{
		svc:      svc,
		keys:     DefaultKeys(),
		themeIdx: 0,
		styles:   NewStyles(th),
		width:    minAppWidth,
		height:   minAppHeight,
		screen:   screenLogin,
		alert: bubbleup.NewAlertModel(50, false, 3*time.Second).
			WithMinWidth(20).
			WithPosition(bubbleup.BottomRightPosition).
			WithUnicodePrefix(),
	}
	a.login = newLoginModel(svc, a.styles)
	a.inv = newInventoryModel(svc, a.styles)
	a.sess = newSessionsModel(svc, a.styles)
	a.settings = newSettingsModel(svc, a.styles)
	a.cmdPalette = newPaletteModel(a.styles)
	a.help = newHelpModel()
	a.sidebar = newSidebarModel()
	a.logs = newLogPane()
	return a
}

// SetSend wires program.Send so async callbacks (device code) can inject msgs.
func (a *App) SetSend(f func(tea.Msg)) { a.send = f }

// setTheme is the single place that switches themes, so every component that
// bakes theme colors into its own state (e.g. the inventory table's styles
// and pre-rendered row glyphs) gets re-synced consistently.
func (a *App) setTheme(themeIdx int) {
	a.themeIdx = themeIdx
	a.styles = NewStyles(Themes[themeIdx])
	// a.login.applyStyles(a.styles)
	a.inv.SetStyles(a.styles)
	a.sess.applyStyles(a.styles)
	a.settings.applyStyles(a.styles)
	a.cmdPalette.applyStyles(a.styles)
}

func (a *App) Init() tea.Cmd {
	// The braille spinner starts ticking immediately for the login/sync UI.
	return tea.Batch(a.login.spinner.Tick, a.alert.Init())
}

func (a *App) toast(key, message string) tea.Cmd {
	if a.overlay != overlayNone || a.tooSmall() {
		return nil
	}
	return a.alert.NewAlertCmd(key, message)
}

func (a *App) notify(key, message string) tea.Cmd {
	isNew := a.sidebar.notifs.add(key, message)
	a.seeNotifsIfVisible()
	if !isNew {
		return nil
	}
	return a.toast(key, message)
}

func (a *App) seeNotifsIfVisible() {
	if a.overlay == overlaySidebar && a.sidebar.tab == sidebarTabNotif {
		a.sidebar.notifs.markAllRead()
	}
}

func (a *App) log(level, msg string) tea.Cmd {
	a.logs.add(level, msg)
	if !a.settings.logAlerts || !a.logs.shouldToast(msg) {
		return nil
	}
	switch level {
	case "WARN":
		return a.toast(bubbleup.WarnKey, msg)
	case "ERR", "ERROR":
		return a.toast(bubbleup.ErrorKey, msg)
	default:
		return a.toast(bubbleup.DebugKey, msg)
	}
}

func tabName(t tab) string {
	switch t {
	case tabInventory:
		return "inventory"
	case tabSessions:
		return "sessions"
	default:
		return "unknown"
	}
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	_, cmd := a.handle(msg)
	out, alertCmd := a.alert.Update(msg)
	a.alert = out.(bubbleup.AlertModel)
	return a, tea.Batch(cmd, alertCmd)
}

func (a *App) handle(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		a.clampPan()
		a.propagateSize()
		return a, nil

	case tea.KeyPressMsg:
		if a.keys.Global.Match(msg) == ActionForceQuit {
			return a, tea.Quit
		}
		if a.tooSmall() {
			a.panTooSmall(msg)
			return a, nil
		}
		if a.overlay != overlayNone {
			return a.updateOverlay(msg)
		}
		if a.screen == screenLogin {
			prev := a.login.step
			m, cmd := a.login.Update(msg, a.send, a.styles, a.keys.Nav)
			a.login = m
			if prev == stepSelect && m.step == stepWaiting {
				return a, tea.Batch(cmd, a.log("INFO", "login started via "+m.selected))
			}
			if prev == stepWaiting && m.step == stepSelect {
				return a, tea.Batch(cmd, a.log("INFO", "login cancelled"))
			}
			return a, cmd
		}
		return a.updateAppKeys(msg)

	case deviceCodeMsg:
		a.login.deviceCode = core.DeviceCode(msg)
		return a, a.log("INFO", "device code issued for "+a.login.selected)

	case loginResultMsg:
		if msg.err != nil {
			a.login.err = msg.err
			a.login.step = stepSelect
			return a, a.log("ERROR", "login failed: "+msg.err.Error())
		}
		a.identity = msg.identityProvider
		a.screen = screenApp

		userInfo, err := a.identity.UserInfo(context.Background())
		if err != nil {
			a.loggedUserLabel = a.styles.Dim.Render("unknown")
			cmds = append(cmds, a.log("ERROR", "user info: "+err.Error()))
			cmds = append(cmds, a.notify(bubbleup.ErrorKey, "error: "+err.Error()))
		} else {
			userName := userInfo["name"]
			// userEmail := userInfo["email"]
			identityProviderId := string(a.identity.ID())
			// a.loggedUser = a.styles.Dim.Render(userName + " (" + userEmail + ") · " + identityProviderId)
			a.loggedUserLabel = a.styles.Dim.Render(userName + " via " + identityProviderId)
			cmds = append(cmds, a.log("INFO", "signed in as "+userName+" via "+identityProviderId))
			cmds = append(cmds, a.notify(bubbleup.InfoKey, "welcome, "+userName))
		}

		for cp, cerr := range msg.cpErrors {
			cmds = append(cmds, a.log("WARN", string(cp)+" connect failed: "+cerr.Error()))
		}

		cmds = append(cmds, a.log("INFO", "syncing cloud providers"))
		cmds = append(cmds, syncProviderCmds(a.svc, false)...)
		cmds = append(cmds, autoRefreshCmd())

		return a, tea.Batch(cmds...)

	case vmsLoadedMsg:
		a.inv.mergeProvider(msg.provider, msg.vms)
		return a, a.log("INFO", string(msg.provider)+" synced ("+itoa(len(msg.vms))+" VMs)")

	case vmsLoadErrMsg:
		provider_str := string(msg.provider)
		if errors.Is(msg.err, core.ErrNotAuthenticated) || errors.Is(msg.err, core.ErrExchange) {
			return a, tea.Batch(
				a.log("WARN", provider_str+" token expired"),
				a.notify(bubbleup.ErrorKey, provider_str+": session expired, VMs not loaded"),
			)
		}
		return a, tea.Batch(
			a.log("WARN", provider_str+": "+msg.err.Error()),
			a.notify(bubbleup.ErrorKey, provider_str+": "+msg.err.Error()),
		)

	case connectErrMsg:
		return a, tea.Batch(
			a.log("ERROR", "connect failed: "+msg.vm),
			a.notify(bubbleup.ErrorKey, "connect failed: "+msg.vm),
		)

	case sessionOpenedMsg:
		return a, tea.Batch(
			a.log("INFO", "session opened: "+msg.spec.VMName),
			execSessionCmd(msg.spec),
		)

	case sessionClosedMsg:
		a.sess.CloseSession(context.Background(), msg.id)
		return a, tea.Batch(
			a.log("INFO", "session closed: "+msg.id),
			a.notify(bubbleup.InfoKey, "session closed"),
		)

	case autoRefreshMsg:
		if a.settings.autoRefresh {
			cmds = append(cmds, a.log("INFO", "auto-refresh"))
			cmds = append(cmds, syncProviderCmds(a.svc, true)...)
		}
		cmds = append(cmds, autoRefreshCmd())
		return a, tea.Batch(cmds...)

	case tokenExpiredMsg:
		return a, tea.Batch(
			a.log("WARN", string(msg.provider)+" token expired"),
			a.notify(bubbleup.ErrorKey, string(msg.provider)+": session expired, VMs not loaded"),
		)

	case providerReconnectOKMsg:
		cmds = append(cmds, a.log("INFO", string(msg.provider)+" reconnected"))
		cmds = append(cmds, a.notify(bubbleup.InfoKey, string(msg.provider)+" reconnected"))
		cmds = append(cmds, syncProviderCmd(a.svc, msg.provider, 0))
		return a, tea.Batch(cmds...)

	case providerReconnectErrMsg:
		return a, tea.Batch(
			a.log("WARN", string(msg.provider)+" reconnect failed: "+msg.err.Error()),
			a.notify(bubbleup.ErrorKey, string(msg.provider)+" reconnect failed: "+msg.err.Error()),
		)
	}

	// Delegate ticks (spinner) and component msgs to the active area.
	return a.delegate(msg)
}

func (a *App) layoutWidth() int  { return max(a.width, minAppWidth) }
func (a *App) layoutHeight() int { return max(a.height, minAppHeight) }

func (a *App) clampPan() {
	a.panX = max(0, min(a.panX, max(0, a.layoutWidth()-max(1, a.width))))
	a.panY = max(0, min(a.panY, max(0, a.layoutHeight()-max(1, a.height))))
}

func (a *App) panTooSmall(msg tea.KeyPressMsg) {
	switch a.keys.Nav.Match(msg) {
	case ActionMoveUp:
		a.panY--
	case ActionMoveDown:
		a.panY++
	case ActionPageUp:
		a.panY -= max(1, a.height-1)
	case ActionPageDown:
		a.panY += max(1, a.height-1)
	case ActionJumpTop:
		a.panY = 0
	case ActionJumpBottom:
		a.panY = a.layoutHeight()
	}
	switch msg.Code {
	case tea.KeyLeft, 'h':
		a.panX--
	case tea.KeyRight, 'l':
		a.panX++
	}
	a.clampPan()
}

func (a *App) View() tea.View {
	var body string
	if a.screen == screenLogin {
		body = a.login.View(a.layoutWidth(), a.layoutHeight())
	} else {
		body = a.appView()
	}

	// Overlays via the Lip Gloss v2 compositor (no manual z-index).
	// The min-size dialog wins over help/quit/palette.
	if a.tooSmall() || a.overlay != overlayNone {
		if a.tooSmall() {
			body = cropBlock(body, a.panX, a.panY, max(1, a.width), max(1, a.height))
		}
		body = a.composeOverlay(body)
	} else if a.screen == screenLogin {
		body = a.alert.Render(body)
	}
	body = cropBlock(body, 0, 0, max(1, a.width), max(1, a.height))

	v := tea.NewView(body)
	v.AltScreen = true
	v.BackgroundColor = a.styles.th.Bg
	v.WindowTitle = "stratus — multi-cloud vm gateway"
	// Ask the terminal to report modifiers (ctrl+shift+t vs ctrl+t).
	v.KeyboardEnhancements.ReportAllKeysAsEscapeCodes = true
	v.KeyboardEnhancements.ReportAlternateKeys = true
	return v
}

func (a *App) appView() string {
	// contentHeight() depends on more than window size (active tab),
	// so re-propagate on every render rather than only on WindowSizeMsg —
	// otherwise a tab switch would leave the converted sub-models' cached
	// table/list sizes stale.
	a.propagateSize()

	top := a.topChromeView()
	bottom := a.bottomChromeView()
	contentW := max(1, a.layoutWidth())
	layoutH := a.layoutHeight()

	var mid string
	switch a.tab {
	case tabInventory:
		mid = a.inv.View()
	case tabSessions:
		mid = a.sess.View()
	}
	if !a.tooSmall() && a.overlay == overlayNone {
		// Toasts sit on the tab body so they never cover the status bar.
		midH := max(1, layoutH-blockHeight(top)-blockHeight(bottom))
		mid = a.alert.Render(padBlock(mid, contentW, midH))
	}
	return stackTopMidBottom(top, mid, bottom, contentW, layoutH)
}
