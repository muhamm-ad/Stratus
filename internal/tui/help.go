package tui

import "charm.land/bubbles/v2/help"

type helpModel struct {
	hm help.Model
}

func newHelpModel() helpModel {
	h := help.New()
	h.ShowAll = true // this overlay is the full panel, not the one-line short bar
	h.SetWidth(58)   // OverlayBox.Width(64) minus its Padding(1,2) horizontal budget
	return helpModel{hm: h}
}

func (m helpModel) View(s Styles, k KeyMap) string {
	// help.Model.View(k) takes the keymap fresh each call rather than baking
	// styles in via a setter, so re-deriving colors from the current theme
	// here (unlike table/list's SetStyles) is the idiomatic, stateless way.
	m.hm.Styles.FullKey = s.Accent
	m.hm.Styles.FullDesc = s.Dim
	m.hm.Styles.FullSeparator = s.Dim
	m.hm.Styles.Ellipsis = s.Dim

	body := m.hm.View(k)
	footer := s.Dim.Render("esc close")
	return s.OverlayBox.Width(64).Render(
		s.Accent.Bold(true).Render("keyboard shortcuts") + "\n\n" + body + "\n\n" + footer,
	)
}
