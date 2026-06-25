package main

import (
	"context"
	"fmt"

	"github.com/muhamm-ad/stratus/config"
	"github.com/muhamm-ad/stratus/core"
	"github.com/muhamm-ad/stratus/providers/aws"
)

// App struct
type App struct {
	ctx context.Context
	registry *core.Registry
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		registry: core.NewRegistry(),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	secs, err := config.Load()
	if err != nil {
		fmt.Printf("config: %v", err)
		return
	}
	if err := aws.Register(a.registry, secs.AWS); err != nil {
		fmt.Printf("aws désactivé : %v", err)
	} else {
		fmt.Printf("aws activé")
	}

	// azure.Register(a.registry, secs.Azure)
	// gcp.Register(a.registry, secs.GCP)
}