package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/muhamm-ad/stratus/cmd/shared"
	"github.com/muhamm-ad/stratus/internal/tui"
)

func main() {
	svc, warnings, err := shared.Init()

	if err != nil {
		fmt.Fprintln(os.Stderr, "stratus:", err)
		os.Exit(1)
	}

	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "stratus: warning: %v\n", w)
	}

	app := tui.New(svc)

	p := tea.NewProgram(app)
	app.SetSend(p.Send)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "stratus:", err)
		os.Exit(1)
	}
}
