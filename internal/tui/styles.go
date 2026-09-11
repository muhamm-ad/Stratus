package tui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Styles holds every reusable lipgloss.Style, derived from a Theme. We rebuild
// this whenever the theme changes so the whole UI recolors in one place.
type Styles struct {
	th Theme

	App                                    lipgloss.Style
	Header                                 lipgloss.Style
	TabActive                              lipgloss.Style
	TabInactive                            lipgloss.Style
	Title                                  lipgloss.Style
	SectionHead                            lipgloss.Style // UPPERCASE dim headers
	StatusBar                              lipgloss.Style
	FilterLine                             lipgloss.Style
	Box                                    lipgloss.Style // rounded border, surface bg (login)
	SidePanel                              lipgloss.Style // left-border-only docked pane (vm detail, sessions)
	Sidebar                                lipgloss.Style // right-docked overlay drawer
	Cursor                                 lipgloss.Style // ▸ cursor (accent)
	Marked                                 lipgloss.Style // warn-tinted marked row
	Dialog                                 lipgloss.Style // overlay chrome: rounded accent border, padded
	DialogTitle                            lipgloss.Style
	DialogBody                             lipgloss.Style
	DialogKey                              lipgloss.Style
	DialogBtn                              lipgloss.Style
	DialogQuitBtn                          lipgloss.Style
	HelpCategory                           lipgloss.Style
	HelpKey                                lipgloss.Style
	HelpDesc                               lipgloss.Style
	CodeBox                                lipgloss.Style // big device code box
	Dim, Accent, OK, Err, Warn, Cyan, Text lipgloss.Style
}

func NewStyles(th Theme) Styles {
	base := lipgloss.NewStyle().Foreground(th.Text)
	return Styles{
		th:     th,
		App:    lipgloss.NewStyle().Background(th.Bg).Foreground(th.Text),
		Header: lipgloss.NewStyle(),
		// Border-based tabs (see chrome.go's tabBorderWithBottom): the active
		// tab's bottom edge is open so it visually merges into the row below,
		// while inactive tabs sit on a flowing "┴" connector — bold + shape
		// carry the active/inactive distinction even on NoColor themes.
		TabActive:   lipgloss.NewStyle().Border(tabActiveBorder, true).BorderForeground(th.Accent).Foreground(th.Accent).Bold(true).Padding(0, 1),
		TabInactive: lipgloss.NewStyle().Border(tabInactiveBorder, true).BorderForeground(th.Border).Foreground(th.Dim).Padding(0, 1),
		Title:       lipgloss.NewStyle().Foreground(th.Accent).Bold(true),
		SectionHead: lipgloss.NewStyle().Foreground(th.Dim).Bold(true), // callers upper-case the text
		StatusBar: lipgloss.NewStyle().
			Background(th.Bg).
			Foreground(th.Dim).
			Padding(0).
			Margin(0),
		FilterLine: lipgloss.NewStyle().Foreground(th.Dim),
		Box:        lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(th.Border).Background(th.Surface).Padding(1, 3),
		SidePanel:  lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, false, false, true).BorderForeground(th.Border).Background(th.Surface).Padding(0, 1),
		Sidebar: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(th.Accent).
			BorderBackground(th.Bg).
			Background(th.Bg).
			Padding(0, 1, 0, 1),
		Cursor: lipgloss.NewStyle().Foreground(th.Accent).Bold(true),
		Marked: lipgloss.NewStyle().Background(blend(th.Warn, th.Bg)),
		Dialog: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(th.Accent).
			BorderBackground(th.Bg).
			Background(th.Bg).
			Padding(1, 4),
		DialogTitle: lipgloss.NewStyle().Foreground(th.Text).Bold(true),
		DialogBody:  lipgloss.NewStyle().Foreground(th.Dim),
		DialogKey:   lipgloss.NewStyle().Foreground(th.Dim),
		DialogBtn: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(th.Border).
			Foreground(th.Text).
			Padding(0, 2).
			Height(1),
		DialogQuitBtn: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(th.Border).
			Foreground(th.Err).
			Padding(0, 2).
			Height(1),
		HelpCategory: lipgloss.NewStyle().Foreground(th.Accent).Bold(true),
		HelpKey:      lipgloss.NewStyle().Foreground(th.Text),
		HelpDesc:     lipgloss.NewStyle().Foreground(th.Dim),
		CodeBox:      lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(th.Accent).Foreground(th.Accent).Bold(true).Padding(0, 2),
		Dim:          lipgloss.NewStyle().Foreground(th.Dim),
		Accent:       lipgloss.NewStyle().Foreground(th.Accent),
		OK:           lipgloss.NewStyle().Foreground(th.OK),
		Err:          lipgloss.NewStyle().Foreground(th.Err),
		Warn:         lipgloss.NewStyle().Foreground(th.Warn),
		Cyan:         lipgloss.NewStyle().Foreground(th.Cyan),
		Text:         base,
	}
}

// blend tints a background with a foreground color for marked rows.
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
