// Package bootstrap assembles the shared Service from the embedded
// configuration. It is the single wiring point used by BOTH entry points (the
// Wails app and the CLI), so neither duplicates construction. It is also where
// the built-in providers are enabled, via the blank import of providers/all —
// the one place to add or remove a provider plugin.
package bootstrap

import (
	"os"

	"github.com/muhamm-ad/stratus/config"
	"github.com/muhamm-ad/stratus/core"
	"github.com/muhamm-ad/stratus/identity/entra"
	"github.com/muhamm-ad/stratus/service"

	// Register the built-in cloud providers (their init() calls RegisterProvider).
	_ "github.com/muhamm-ad/stratus/providers/all"
)

// New builds the shared Service: it constructs the Entra identity provider and
// every registered + configured connector. The returned warnings list providers
// skipped because their config section is incomplete (the app still runs with
// the others); a non-nil error means a fatal problem (bad config.json or a
// missing/invalid identity section).
func New() (svc *service.Service, warnings []error, err error) {
	secs, err := config.Load()
	if err != nil {
		return nil, nil, err
	}
	entraCfg, err := entra.ParseConfig(secs.IdentitySection("entra"), os.Getenv)
	if err != nil {
		return nil, nil, err
	}
	idp := entra.New(entraCfg)

	reg, warnings := core.BuildAll(core.Factories(), secs.Providers)
	return service.New(idp, reg), warnings, nil
}
