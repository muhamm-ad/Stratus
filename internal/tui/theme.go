package tui

import (
	"image/color"

	"charm.land/lipgloss/v2"
	"github.com/muhamm-ad/stratus/internal/core"
)

// Theme is a color palette. Stratus ships four themes that the user cycles
// with the 't' key. Fixed palettes use hex colors; the "terminal" theme uses
// lipgloss.NoColor and standard ANSI colors so the emulator's own theme shows
// through.
type Theme struct {
	Name    string
	Bg      color.Color
	Surface color.Color
	Surface2 color.Color
	Border  color.Color
	Text    color.Color
	Dim     color.Color
	Accent  color.Color
	OK      color.Color
	Err     color.Color
	Warn    color.Color
	Cyan    color.Color
}

// SpinnerFrames are the braille frames from the HTML mockup.
var SpinnerFrames = []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}

// Provider brand colors (independent of theme), matching the mockup dots.
var (
	AWSColor   = lipgloss.Color("#FF9900")
	AzureColor = lipgloss.Color("#3B9EFF")
	GCPColor   = lipgloss.Color("#4285F4")
)

// Themes is the ordered, cyclable list. Index 0 (charm) is the default.
var Themes = []Theme{
	{
		Name:    "charm",
		Bg:      lipgloss.Color("#191426"),
		Surface: lipgloss.Color("#221c33"),
		Surface2: lipgloss.Color("#2f2749"),
		Border:  lipgloss.Color("#453a63"),
		Text:    lipgloss.Color("#e9e4f7"),
		Dim:     lipgloss.Color("#8d84ad"),
		Accent:  lipgloss.Color("#bd93f9"),
		OK:      lipgloss.Color("#50fa7b"),
		Err:     lipgloss.Color("#ff5555"),
		Warn:    lipgloss.Color("#f1fa8c"),
		Cyan:    lipgloss.Color("#8be9fd"),
	},
	{
		Name:    "stratus",
		Bg:      lipgloss.Color("#0F172A"),
		Surface: lipgloss.Color("#1E293B"),
		Surface2: lipgloss.Color("#334155"),
		Border:  lipgloss.Color("#334155"),
		Text:    lipgloss.Color("#F1F5F9"),
		Dim:     lipgloss.Color("#94A3B8"),
		Accent:  lipgloss.Color("#818CF8"),
		OK:      lipgloss.Color("#4ADE80"),
		Err:     lipgloss.Color("#F87171"),
		Warn:    lipgloss.Color("#FBBF24"),
		Cyan:    lipgloss.Color("#67E8F9"),
	},
	{
		Name:    "mono",
		Bg:      lipgloss.Color("#060606"),
		Surface: lipgloss.Color("#0e0e0e"),
		Surface2: lipgloss.Color("#1e1e1e"),
		Border:  lipgloss.Color("#2e2e2e"),
		Text:    lipgloss.Color("#d8d8d8"),
		Dim:     lipgloss.Color("#6e6e6e"),
		Accent:  lipgloss.Color("#4af626"),
		OK:      lipgloss.Color("#4af626"),
		Err:     lipgloss.Color("#ff5f56"),
		Warn:    lipgloss.Color("#f3bf4f"),
		Cyan:    lipgloss.Color("#7adfe0"),
	},
	{
		Name:     "terminal",
		Bg:       lipgloss.NoColor{},
		Surface:  lipgloss.NoColor{},
		Surface2: lipgloss.BrightBlack,
		Border:   lipgloss.BrightBlack,
		Text:     lipgloss.NoColor{},
		Dim:      lipgloss.BrightBlack,
		Accent:   lipgloss.BrightBlue,
		OK:       lipgloss.Green,
		Err:      lipgloss.Red,
		Warn:     lipgloss.Yellow,
		Cyan:     lipgloss.Cyan,
	},
}

// ProviderColor maps a provider id to its brand color.
func ProviderColor(p core.CloudProviderID) color.Color {
	switch string(p) {
	case "aws":
		return AWSColor
	case "azure":
		return AzureColor
	case "gcp":
		return GCPColor
	}
	return lipgloss.Color("#888888")
}
