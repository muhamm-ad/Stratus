package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/muhamm-ad/stratus/service"
	"github.com/muhamm-ad/stratus/internal/tui"
)


// BuildTUIGateway assembles the Gateway consumed by the TUI. In production this
// wires the real service (backed by providers/all self-registered plugins). For
// the standalone demo build it returns the in-memory mock so
// `go run ./cmd/stratus-tui` works with zero config.
func BuildTUIGateway() (service.Gateway, error) {
	if os.Getenv("STRATUS_REAL") == "" {
		return service.mock.NewService(), nil // demo default
	}
	// TODO: return the real service, e.g.:
	//   _ = all.Register()               // blank-import side effects already ran
	//   svc, err := service.New(cfg)     // depends only on core
	//   return svc, err
	svc, err := service.NewService()
	return svc, err
}

func main() {
	// bootstrap is the single assembly point shared by both entry points
	// (Wails desktop + this TUI). It wires providers→service and returns a
	// service.Gateway. For the standalone demo it returns the mock gateway.
	gw, err := service.BuildTUIGateway()
	if err != nil {
		fmt.Fprintln(os.Stderr, "stratus:", err)
		os.Exit(1)
	}

	app := tui.New(gw)
	p := tea.NewProgram(app)
	app.SetSend(p.Send) // let async login callbacks inject device-code msgs
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "stratus:", err)
		os.Exit(1)
	}
}
