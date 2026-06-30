package main

import (
	"context"
	"fmt"
	"os"

	"github.com/muhamm-ad/stratus/config"
	"github.com/muhamm-ad/stratus/core"
	"github.com/muhamm-ad/stratus/identity/entra"

	// Enable the built-in providers (registers their factories via init()).
	_ "github.com/muhamm-ad/stratus/providers/all"
)

type App struct {
	ctx      context.Context
	identity core.IdentityProvider
	registry *core.Registry
}

func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	secs, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("config: %v", err))
	}

	// Single identity provider (Entra) — the only thing that opens a browser.
	entraCfg, err := entra.ParseConfig(secs.IdentitySection("entra"), os.Getenv)
	if err != nil {
		panic(fmt.Sprintf("identity: %v", err))
	}
	a.identity = entra.New(entraCfg)

	// Build every registered + configured provider
	reg, errs := core.BuildAll(core.Factories(), secs.Providers)
	for _, e := range errs {
		fmt.Printf("provider unavailable: %v\n", e) // incomplete config → skipped
	}
	a.registry = reg
}

// ── Methods bound to the React frontend ─────────────────────────────────────

func (a *App) Login() error          { return a.identity.Login(a.ctx) }
func (a *App) IsAuthenticated() bool { return a.identity.IsAuthenticated() }
func (a *App) Logout() error         { return a.identity.Logout(a.ctx) }

func (a *App) Providers() []string {
	ids := a.registry.IDs()
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = string(id)
	}
	return out
}

// ConnectProvider derives a provider's credentials from the Entra session —
// silent, no browser. Call after Login().
func (a *App) ConnectProvider(id string) error {
	c, ok := a.registry.Get(core.ProviderID(id))
	if !ok {
		return fmt.Errorf("unknown provider: %q", id)
	}
	return c.Authenticate(a.ctx, a.identity)
}
