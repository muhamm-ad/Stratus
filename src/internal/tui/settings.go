package tui

// settingsModel renders providers, auto-refresh, theme, and LOCAL CLI DETECTION.
type settingsModel struct {
	gw          service.Gateway
	cursor      int
	autoRefresh bool
	clis        []service.CLIStatus
}

func newSettingsModel(gw service.Gateway) settingsModel {
	return settingsModel{gw: gw, autoRefresh: true, clis: gw.DetectCLIs()}
}

func (m settingsModel) View(s Styles, w, h int) string {
	head := s.SectionHead.Render("PROVIDERS & PREFERENCES · j/k move · ⏎ toggle/cycle")
	// ... provider rows (connected · <identity> in green / session expired in red /
	//     ⣾ authorizing… in warn / disconnected in dim) ...
	// auto-refresh row: "[on] every 60s" / "[off]"
	// theme row: "charm (charm · stratus · mono)"
	cliHead := s.SectionHead.Render("LOCAL CLI DETECTION")
	var cliRows []string
	for _, c := range m.clis {
		if c.Detected {
			cliRows = append(cliRows, s.OK.Render("✓ ")+s.Text.Render(c.Name+" detected"))
		} else {
			cliRows = append(cliRows, s.Err.Render("✗ ")+s.Text.Render(c.Name+" missing — "+c.Hint))
		}
	}
	_ = head; _ = cliHead
	// JoinVertical(...) omitted for brevity in the report.
	return renderSettings(s, w, head, cliHead, cliRows, m)
}
