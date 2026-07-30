package tui

import (
	"context"
	"errors"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/muhamm-ad/stratus/internal/core"
	"github.com/muhamm-ad/stratus/internal/service"
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
	svc  *service.Service
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
	identity  core.IdentityProvider

	searchMode bool
	searchBuf  string

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
		screen:   screenLogin,
	}
	a.login = newLoginModel(svc, a.styles)
	a.inv = newInventoryModel(svc, a.styles)
	a.sess = newSessionsModel(svc, a.styles)
	a.audit = newAuditModel(svc, a.styles)
	a.settings = newSettingsModel(svc, a.styles)
	a.palette = newPaletteModel(a.styles)
	a.help = newHelpModel()
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
	a.audit.applyStyles(a.styles)
	a.sess.applyStyles(a.styles)
	// a.settings.applyStyles(a.styles)
	a.palette.applyStyles(a.styles)
}

// bannerLines derives one error banner per provider currently in
// CloudProviderStatusError, in stable CloudProvidersIDs order — so
// reconnecting a provider actually makes its banner disappear.
func (a *App) bannerLines() []string {
	var out []string
	for _, cp := range a.svc.GetCloudProvidersIDs() {
		if a.svc.GetCloudProviderStatus(cp) == core.CloudProviderStatusError {
			out = append(out, "▲ "+string(cp)+": session expired, VMs not loaded — press R to reconnect")
		}
	}
	return out
}

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
		if s := msg.String(); s == "ctrl+c" {
			return a, tea.Quit
		}
		if a.overlay != overlayNone {
			return a.updateOverlay(msg)
		}
		if a.screen == screenLogin {
			m, cmd := a.login.Update(msg, a.send, a.styles)
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
			a.login.step = stepSelect
			return a, nil
		}
		a.identity = msg.identityProvider
		a.screen = screenApp

		userInfo, err := a.identity.UserInfo(context.Background())
		if err != nil {
			a.loggedUserLabel = a.styles.Dim.Render("unknown")
			a.flash, a.flashKind = "error: "+err.Error(), "err"
			// 	return a, flashClearCmd()
		} else {
			userName := userInfo["name"]
			// userEmail := userInfo["email"]
			identityProviderId := string(a.identity.ID())
			// a.loggedUser = a.styles.Dim.Render(userName + " (" + userEmail + ") · " + identityProviderId)
			a.loggedUserLabel = a.styles.Dim.Render(userName + " via " + identityProviderId)
			a.flash, a.flashKind = "welcome, "+userName+" — signed in via "+identityProviderId, "ok"
		}

		for cp, cerr := range msg.cpErrors {
			a.logs.add("WARN", string(cp)+" connect failed: "+cerr.Error())
		}

		cmds = append(cmds, flashClearCmd())
		cmds = append(cmds, syncProviderCmds(a.svc, false)...)
		cmds = append(cmds, autoRefreshCmd())

		return a, tea.Batch(cmds...)

	case vmsLoadedMsg:
		a.inv.mergeProvider(msg.provider, msg.vms)
		a.logs.add("INFO", string(msg.provider)+" synced ("+itoa(len(msg.vms))+" VMs)")
		return a, nil

	case vmsLoadErrMsg:
		provider_str := string(msg.provider)
		if errors.Is(msg.err, core.ErrNotAuthenticated) || errors.Is(msg.err, core.ErrExchange) {
			a.logs.add("WARN", provider_str+" token expired")
		} else {
			a.logs.add("WARN", provider_str+": "+msg.err.Error())
		}
		return a, nil

	case connectErrMsg:
		a.flash, a.flashKind = "connect failed: "+msg.vm, "err"
		return a, flashClearCmd()

	case sessionOpenedMsg:
		return a, execSessionCmd(msg.spec)

	case sessionClosedMsg:
		a.sess.CloseSession(context.Background(), msg.id)
		a.logs.add("INFO", "session closed: "+msg.id)
		a.flash, a.flashKind = "session closed", "ok"
		return a, flashClearCmd()

	case flashClearMsg:
		a.flash = a.loggedUserLabel
		return a, nil

	case autoRefreshMsg:
		if a.settings.autoRefresh {
			cmds = append(cmds, syncProviderCmds(a.svc, true)...)
		}
		cmds = append(cmds, autoRefreshCmd())
		return a, tea.Batch(cmds...)

	case tokenExpiredMsg:
		a.logs.add("WARN", string(msg.provider)+" token expired")
		return a, nil

	case providerReconnectOKMsg:
		a.flash, a.flashKind = string(msg.provider)+" reconnected", "ok"
		cmds = append(cmds, flashClearCmd())
		cmds = append(cmds, syncProviderCmd(a.svc, msg.provider, 0))
		return a, tea.Batch(cmds...)

	case providerReconnectErrMsg:
		a.flash, a.flashKind = string(msg.provider)+" reconnect failed: "+msg.err.Error(), "err"
		a.logs.add("WARN", string(msg.provider)+" reconnect failed: "+msg.err.Error())
		return a, flashClearCmd()
	}

	// Delegate ticks (spinner) and component msgs to the active area.
	return a.delegate(msg)
}

func (a *App) View() tea.View {
	var body string
	if a.screen == screenLogin {
		body = a.login.View(a.width, a.height)
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
	// v.MouseMode = tea.MouseModeAllMotion
	return v
}

func (a *App) composeOverlay(background string) string {
	var fg string
	switch a.overlay {
	case overlayPalette:
		fg = a.palette.View()
	case overlayHelp:
		fg = a.help.View(a.styles, a.keys)
	case overlayConfirmQuit:
		fg = a.confirmQuitView()
	default:
		return background
	}

	fgW, fgH := lipgloss.Size(fg)
	x := (a.width - fgW) / 2
	y := (a.height - fgH) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	comp := lipgloss.NewCompositor(
		lipgloss.NewLayer(background),        // z 0
		lipgloss.NewLayer(fg).X(x).Y(y).Z(1), // z 1: floats on top, centered
	)
	return lipgloss.NewCanvas(a.width, a.height).Compose(comp).Render()
}

func (a *App) confirmQuitView() string {
	title := a.styles.Err.Bold(true).Render("sign out of stratus?")
	body := a.styles.Dim.Render("you'll need to re-authenticate with your identity\nprovider next time you start stratus.")
	footer := a.styles.Dim.Render("⏎/y confirm · esc/n cancel")
	return a.styles.ModalBox.Render(title + "\n\n" + body + "\n\n" + footer)
}

func (a *App) appView() string {
	// contentHeight() depends on more than window size (active tab, showLogs,
	// banner count), so re-propagate on every render rather than only on
	// WindowSizeMsg — otherwise a tab switch or banner append would leave the
	// converted sub-models' cached table/list sizes stale.
	a.propagateSize()

	topChrome := a.topChromeView()
	var mid string
	switch a.tab {
	case tabInventory:
		mid = a.inv.View()
	case tabSessions:
		mid = a.sess.View()
	case tabAudit:
		mid = a.audit.View()
	case tabSettings:
		mid = a.settings.View(a.width, a.contentHeight(), a.themeIdx)
	}
	parts := []string{topChrome}
	// if a.tab != tabSettings {
	// 	parts = append(parts, a.filterLineView())
	// }
	for _, b := range a.bannerLines() {
		parts = append(parts, a.styles.ErrorBanner.Width(a.width).Render(b))
	}
	parts = append(parts, mid)
	if a.showLogs {
		parts = append(parts, a.logs.View(a.styles, a.width))
	}
	parts = append(parts, a.bottomChromeView())
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}
