package tui

import (
	"fmt"
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

	x, y := modalOrigin(fg, a.width, a.height)
	comp := lipgloss.NewCompositor(
		lipgloss.NewLayer(blurContent(background, a.styles.Dim)),
		lipgloss.NewLayer(fg).X(x).Y(y).Z(1),
	)
	return lipgloss.NewCanvas(max(1, a.width), max(1, a.height)).Compose(comp).Render()
}

func modalOrigin(modal string, w, h int) (x, y int) {
	return max(0, (w-lipgloss.Width(modal))/2), max(0, (h-lipgloss.Height(modal))/2)
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
	}
	return s.Dialog.Align(lipgloss.Center).Render(
		lipgloss.JoinVertical(lipgloss.Center, lines...),
	)
}

func (a *App) confirmQuitView() string {
	s := a.styles
	lines := []string{
		s.DialogTitle.Render("Sign out of Stratus?"),
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
