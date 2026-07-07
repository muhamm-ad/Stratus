package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/muhamm-ad/stratus/internal/tui"
	"github.com/muhamm-ad/stratus/service"
)

// BuildTUIGateway wires the real service (providers via service.Init). Exits
// with an error when config.json is missing or invalid — no mock fallback.
func BuildTUIGateway() (tui.Gateway, error) {
	svc, warnings, err := service.Init()
	if err != nil {
		return nil, err
	}
	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "stratus: warning: %v\n", w)
	}
	return tui.NewGateway(svc), nil
}

func main() {
	gw, err := BuildTUIGateway()
	if err != nil {
		fmt.Fprintln(os.Stderr, "stratus:", err)
		os.Exit(1)
	}

	app := tui.New(gw)
	p := tea.NewProgram(app)
	app.SetSend(p.Send)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "stratus:", err)
		os.Exit(1)
	}
}
