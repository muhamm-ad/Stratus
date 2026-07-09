package tui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Styles holds every reusable lipgloss.Style, derived from a Theme. We rebuild
// this whenever the theme changes so the whole UI recolors in one place.
type Styles struct {
	th Theme

	App          lipgloss.Style
	Header       lipgloss.Style
	TabActive    lipgloss.Style
	TabInactive  lipgloss.Style
	Title        lipgloss.Style
	SectionHead  lipgloss.Style // UPPERCASE dim headers
	StatusBar    lipgloss.Style
	FilterLine   lipgloss.Style
	Box          lipgloss.Style // rounded border, surface bg (login/overlays)
	Cursor       lipgloss.Style // ▸ cursor row bg (surface2)
	Marked       lipgloss.Style // warn-tinted marked row
	ErrorBanner  lipgloss.Style
	OverlayBox   lipgloss.Style
	ModalBox     lipgloss.Style
	CodeBox      lipgloss.Style // big device code box
	Dim, Accent, OK, Err, Warn, Cyan, Text lipgloss.Style
}

func NewStyles(th Theme) Styles {
	base := lipgloss.NewStyle().Foreground(th.Text)
	tabFG := th.Bg
	if _, ok := th.Bg.(lipgloss.NoColor); ok {
		tabFG = lipgloss.Black
	}
	return Styles{
		th:          th,
		App:         lipgloss.NewStyle().Background(th.Bg).Foreground(th.Text),
		Header:      lipgloss.NewStyle().Background(th.Surface).Foreground(th.Text),
		TabActive:   lipgloss.NewStyle().Background(th.Accent).Foreground(tabFG).Bold(true).Padding(0, 1),
		TabInactive: lipgloss.NewStyle().Foreground(th.Dim).Padding(0, 1),
		Title:       lipgloss.NewStyle().Foreground(th.Accent).Bold(true),
		SectionHead: lipgloss.NewStyle().Foreground(th.Dim).Bold(true), // callers upper-case the text
		StatusBar:   lipgloss.NewStyle().Background(th.Surface).Foreground(th.Dim),
		FilterLine:  lipgloss.NewStyle().Foreground(th.Dim),
		Box:         lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(th.Border).Background(th.Surface).Padding(1, 3),
		Cursor:      lipgloss.NewStyle().Background(th.Surface2),
		Marked:      lipgloss.NewStyle().Background(blend(th.Warn, th.Bg)),
		ErrorBanner: lipgloss.NewStyle().Foreground(th.Err).Background(blend(th.Err, th.Bg)),
		OverlayBox:  lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(th.Accent).Background(th.Surface).Padding(1, 2),
		ModalBox:    lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(th.Err).Background(th.Surface).Padding(1, 2),
		CodeBox:     lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(th.Accent).Foreground(th.Accent).Bold(true).Padding(0, 2),
		Dim:         lipgloss.NewStyle().Foreground(th.Dim),
		Accent:      lipgloss.NewStyle().Foreground(th.Accent),
		OK:          lipgloss.NewStyle().Foreground(th.OK),
		Err:         lipgloss.NewStyle().Foreground(th.Err),
		Warn:        lipgloss.NewStyle().Foreground(th.Warn),
		Cyan:        lipgloss.NewStyle().Foreground(th.Cyan),
		Text:        base,
	}
}

// blend tints a background with a foreground color for marked rows / banners.
func blend(fg, bg color.Color) color.Color {
	if _, ok := bg.(lipgloss.NoColor); ok {
		return lipgloss.BrightBlack
	}
	colors := lipgloss.Blend1D(2, fg, bg)
	if len(colors) > 1 {
		return colors[1]
	}
	return bg
}
