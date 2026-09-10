package tui

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	quitBtnQuit   = 0
	quitBtnCancel = 1
	quitBtnGap    = 2

	minAppWidth  = 95
	minAppHeight = 30
)

// blurContent dims the page behind a modal: strip colors and paint it faint so
// the dialog reads as the only interactive layer.
func blurContent(s string, dim lipgloss.Style) string {
	return dim.Faint(true).Render(ansi.Strip(s))
}

func (a *App) tooSmall() bool {
	return a.width < minAppWidth || a.height < minAppHeight
}

func (a *App) composeOverlay(background string) string {
	blurred := blurContent(background, a.styles.Dim)
	if a.overlay == overlaySidebar && !a.tooSmall() {
		return a.composeSidebarOverlay(blurred)
	}

	var fg string
	switch {
	case a.tooSmall():
		fg = a.minSizeDialog()
	case a.overlay == overlayPalette:
		fg = a.cmdPalette.View(a.width, a.height)
	case a.overlay == overlayHelp:
		fg = a.help.View(a.styles, a.keys, a.width, a.height)
	case a.overlay == overlayConfirmQuit:
		fg = a.confirmQuitView()
	default:
		return background
	}

	fg = opaqueOverlay(fg, a.styles.th.Bg)
	x, y := modalOrigin(fg, a.width, a.height)
	comp := lipgloss.NewCompositor(
		lipgloss.NewLayer(blurred),
		lipgloss.NewLayer(fg).X(x).Y(y).Z(1),
	)
	return lipgloss.NewCanvas(max(1, a.width), max(1, a.height)).Compose(comp).Render()
}

func (a *App) composeSidebarOverlay(blurred string) string {
	panel := opaqueOverlay(a.sidebar.View(a.styles, a.keys, a.height, a.logs, a.settings, a.themeIdx), a.styles.th.Bg)
	handle := sidebarHandleView(a.styles, true)
	px, py := sidebarOrigin(panel, a.width)
	hx := max(0, px-lipgloss.Width(handle))
	hy := max(0, lipgloss.Height(a.tabBarView())-1)
	comp := lipgloss.NewCompositor(
		lipgloss.NewLayer(blurred),
		lipgloss.NewLayer(panel).X(px).Y(py).Z(1),
		lipgloss.NewLayer(handle).X(hx).Y(hy).Z(2),
	)
	return lipgloss.NewCanvas(max(1, a.width), max(1, a.height)).Compose(comp).Render()
}

// opaqueOverlay restamps a composited layer as a complete rectangle. Lip Gloss
// clears a layer's bounding box before drawing; cells that aren't restamped
// (ragged lines, trailing newlines) punch holes through the overlay fill.
func opaqueOverlay(content string, bg color.Color) string {
	w := lipgloss.Width(content)
	h := lipgloss.Height(content)
	if w < 1 || h < 1 {
		return content
	}
	st := lipgloss.NewStyle()
	if _, ok := bg.(lipgloss.NoColor); !ok {
		st = st.Background(bg)
	}
	return boxNoWrap(st, content, w, h)
}

func modalOrigin(modal string, w, h int) (x, y int) {
	return max(0, (w-lipgloss.Width(modal))/2), max(0, (h-lipgloss.Height(modal))/2)
}

func sidebarOrigin(modal string, w int) (x, y int) {
	return max(0, w-lipgloss.Width(modal)), 0
}

func (a *App) minSizeDialog() string {
	s := a.styles
	required := fmt.Sprintf("%d × %d", minAppWidth, minAppHeight)
	wStyle, hStyle := s.DialogTitle, s.DialogTitle
	if a.width < minAppWidth {
		wStyle = s.Err.Bold(true)
	}
	if a.height < minAppHeight {
		hStyle = s.Err.Bold(true)
	}
	current := lipgloss.JoinHorizontal(lipgloss.Center,
		wStyle.Render(fmt.Sprintf("%d", a.width)),
		s.DialogTitle.Render(" × "),
		hStyle.Render(fmt.Sprintf("%d", a.height)),
	)
	lines := []string{
		s.DialogTitle.Render("Window too small"),
		"",
		s.DialogBody.Render("Minimum required size:"),
		s.DialogTitle.Render(required),
		"",
		s.DialogBody.Render("Current size:"),
		current,
		"",
		s.DialogKey.Render("Resize the terminal to continue"),
		s.DialogKey.Render("↑↓←→ or hjkl to pan"),
	}
	return s.Dialog.Align(lipgloss.Center).Render(
		lipgloss.JoinVertical(lipgloss.Center, lines...),
	)
}

func (a *App) confirmQuitView() string {
	s := a.styles
	lines := []string{
		s.DialogTitle.Render("Sign out of Stratus?"),
		"",
		s.DialogBody.Render("You'll need to re-authenticate with your identity"),
		s.DialogBody.Render("provider next time you start Stratus."),
		"",
		a.quitButtonsRow(),
	}
	return s.Dialog.Align(lipgloss.Center).Render(
		lipgloss.JoinVertical(lipgloss.Center, lines...),
	)
}

func (a *App) quitButtonsRow() string {
	quit := a.dialogButton("Sign out", a.quitFocus == quitBtnQuit, true)
	cancel := a.dialogButton("Cancel", a.quitFocus == quitBtnCancel, false)
	return lipgloss.JoinHorizontal(lipgloss.Center, quit, strings.Repeat(" ", quitBtnGap), cancel)
}

func (a *App) dialogButton(label string, focused, danger bool) string {
	style := a.styles.DialogBtn
	if danger {
		style = a.styles.DialogQuitBtn
	}
	if focused {
		style = style.BorderForeground(a.styles.th.Accent)
	}
	return style.Render(label)
}
