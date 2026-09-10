package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestCtrlShiftMatchesThemeAndLogs(t *testing.T) {
	k := DefaultKeys()

	cases := []struct {
		name string
		msg  tea.KeyPressMsg
		want Action
	}{
		{
			"ctrl+shift+t mods",
			tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl | tea.ModShift},
			ActionTheme,
		},
		{
			"ctrl+T encodes shift as case",
			tea.KeyPressMsg{Code: 'T', Mod: tea.ModCtrl},
			ActionTheme,
		},
		{
			"ctrl+shift+t with associated text",
			tea.KeyPressMsg{Text: "T", Code: 't', Mod: tea.ModCtrl | tea.ModShift},
			ActionTheme,
		},
		{
			"plain ctrl+t is not theme",
			tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl},
			ActionNone,
		},
		{
			"ctrl+shift+l mods",
			tea.KeyPressMsg{Code: 'l', Mod: tea.ModCtrl | tea.ModShift},
			ActionLogs,
		},
		{
			"ctrl+L encodes shift as case",
			tea.KeyPressMsg{Code: 'L', Mod: tea.ModCtrl},
			ActionLogs,
		},
		{
			"shift+g is jump bottom",
			tea.KeyPressMsg{Code: 'g', Mod: tea.ModShift},
			ActionJumpBottom,
		},
		{
			"g is jump top",
			tea.KeyPressMsg{Code: 'g'},
			ActionJumpTop,
		},
		{
			"shift+tab is overlay prev",
			tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift},
			ActionOverlayPrev,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := k.Global.Match(tc.msg)
			if got == ActionNone {
				got = k.Nav.Match(tc.msg)
			}
			if got == ActionNone {
				got = k.Overlay.Match(tc.msg)
			}
			if got != tc.want {
				t.Fatalf("got %d want %d (string=%q keystroke=%q)",
					got, tc.want, tc.msg.String(), tc.msg.Keystroke())
			}
		})
	}
}

func TestCanonicalKeyNameFoldsShiftedLetter(t *testing.T) {
	if got, want := canonicalKeyName("ctrl+shift+t"), "ctrl+shift+t"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if got, want := canonicalKeyName("ctrl+T"), "ctrl+shift+t"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if got, want := canonicalKeyName("ctrl+t"), "ctrl+t"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if got, want := canonicalKeyName("G"), "shift+g"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
