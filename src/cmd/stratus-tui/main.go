// Command stratus is the full-screen terminal interface (TUI) for Stratus.
// It assembles the shared service via bootstrap (the same wiring the desktop
// app uses) and runs the Bubble Tea program.
//
//	go run ./cmd/stratus-tui
//
// Configure identity + providers in config/config.json, or via the STRATUS_*
// environment variables.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/muhamm-ad/stratus/bootstrap"
)

func main() {
	svc, warnings, err := bootstrap.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "stratus: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(New(svc, warnings), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "stratus: %v\n", err)
		os.Exit(1)
	}
}