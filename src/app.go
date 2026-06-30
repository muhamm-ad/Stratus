package main

import (
	"context"
	"fmt"

	"github.com/muhamm-ad/stratus/bootstrap"
	"github.com/muhamm-ad/stratus/core"
	"github.com/muhamm-ad/stratus/service"
)

// App is the Wails-bound desktop adapter. It holds NO orchestration logic — it
// delegates everything to *service.Service, the same shared logic the CLI
// (cmd/stratus-cli) uses. Both UIs differ only in presentation.
type App struct {
	ctx context.Context
	svc *service.Service
}

func NewApp() *App { return &App{} }

// startup is called by Wails with the application context.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	svc, warnings, err := bootstrap.New() // same assembly the CLI calls
	if err != nil {
		panic(fmt.Sprintf("bootstrap: %v", err))
	}
	for _, w := range warnings {
		fmt.Printf("provider unavailable: %v\n", w) // incomplete config → skipped
	}
	a.svc = svc
}

// ── Methods bound to the React frontend (thin delegation) ───────────────────

func (a *App) Login() error          { return a.svc.Login(a.ctx) }
func (a *App) IsAuthenticated() bool { return a.svc.IsAuthenticated() }
func (a *App) Logout() error         { return a.svc.Logout(a.ctx) }

func (a *App) Providers() []string {
	ids := a.svc.Providers()
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = string(id)
	}
	return out
}

// ConnectProvider derives a provider's credentials from the Entra session —
// silent, no browser. Call after Login().
func (a *App) ConnectProvider(id string) error {
	return a.svc.Connect(a.ctx, core.ProviderID(id))
}
