// Package service is the shared application layer of Stratus. Both user
// interfaces — the Wails/React desktop UI and the CLI — call into it; neither
// duplicates the orchestration. It depends ONLY on core interfaces; the
// concrete identity provider and connectors are injected (see package
// bootstrap), so this layer never imports config, entra, or any provider.
package service

import (
	"context"
	"fmt"

	"github.com/muhamm-ad/stratus/core"
)

// Service exposes the high-level operations both UIs need.
type Service struct {
	identity core.IdentityProvider
	registry *core.Registry
}

// New builds a Service from an identity provider and a connector registry.
func New(identity core.IdentityProvider, registry *core.Registry) *Service {
	return &Service{identity: identity, registry: registry}
}

// Login performs the single interactive Entra sign-in (browser once).
func (s *Service) Login(ctx context.Context) error { return s.identity.Login(ctx) }

// IsAuthenticated reports whether the Entra session is valid.
func (s *Service) IsAuthenticated() bool { return s.identity.IsAuthenticated() }

// Logout clears the Entra session.
func (s *Service) Logout(ctx context.Context) error { return s.identity.Logout(ctx) }

// Providers lists the registered provider IDs.
func (s *Service) Providers() []core.ProviderID { return s.registry.IDs() }

// Connect derives a provider's credentials from the Entra session (silent
// exchange — no browser). Call after Login.
func (s *Service) Connect(ctx context.Context, id core.ProviderID) error {
	c, ok := s.registry.Get(id)
	if !ok {
		return fmt.Errorf("service: unknown provider %q", id)
	}
	return c.Authenticate(ctx, s.identity)
}

// ListInstances returns connectable VMs for one account on a provider. (Phase 3.)
func (s *Service) ListInstances(ctx context.Context, id core.ProviderID, account string) ([]core.Instance, error) {
	c, ok := s.registry.Get(id)
	if !ok {
		return nil, fmt.Errorf("service: unknown provider %q", id)
	}
	return c.ListInstances(ctx, account)
}

// OpenSession opens an interactive session to one instance. (Phase 4.)
func (s *Service) OpenSession(ctx context.Context, id core.ProviderID, req core.ConnectRequest) (core.Session, error) {
	c, ok := s.registry.Get(id)
	if !ok {
		return nil, fmt.Errorf("service: unknown provider %q", id)
	}
	return c.Connect(ctx, req)
}
