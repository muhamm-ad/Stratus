package tui

import (
	"charm.land/lipgloss/v2"
)

// composeOverlay centers the current overlay box over the background using the
// Lip Gloss v2 compositor. This is the key v2 win: overlays are first-class,
// so we don't hand-roll the line-by-line slicing overlay hack that v1 needed.
func (a *App) composeOverlay(background string) string {
	var fg string
	switch a.overlay {
	case overlayPalette:
		fg = a.palette.View(a.styles)
	case overlayHelp:
		fg = a.help.View(a.styles)
	case overlayConfirmQuit:
		fg = a.confirmQuitView()
	default:
		return background
	}

	fgW, fgH := lipgloss.Size(fg)
	x := (a.width - fgW) / 2
	y := (a.height - fgH) / 2
	if x < 0 { x = 0 }
	if y < 0 { y = 0 }

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
