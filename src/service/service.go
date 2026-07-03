// Package service is the shared application layer of Stratus. Both user
// interfaces — the Wails/React desktop UI and the CLI/TUI — call into it;
// neither duplicates orchestration. It depends ONLY on core interfaces; the
// concrete identity providers and connectors are injected see import statements.
package service

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/muhamm-ad/stratus/config"
	"github.com/muhamm-ad/stratus/core"
	"github.com/muhamm-ad/stratus/identity/oidc"

	// Register the built-in cloud providers (their init() calls RegisterProvider).
	_ "github.com/muhamm-ad/stratus/providers/all"
)

// Service exposes the high-level operations both UIs need. It holds several
// configured identity providers (all OIDC) and the cloud connectors. Exactly
// one identity provider is "active" at a time — the one the user signed in with.
type Service struct {
	idps          map[string]core.IdentityProvider
	activeIdpName string // config key of the signed-in provider; "" = none
	registry      *core.Registry
}

// New builds a Service from the configured identity providers and the connector
// registry. When exactly one identity provider is configured it becomes active
// automatically (no choice to make).
func NewService(idps map[string]core.IdentityProvider, registry *core.Registry) *Service {
	s := &Service{idps: idps, registry: registry}
	if len(idps) == 1 {
		for name := range idps {
			s.activeIdpName = name
		}
	}
	return s
}

// IdentityProviders lists the configured identity-provider names (e.g. "entra",
// "okta") for the single-sign-on choice, sorted.
func (s *Service) IdentityProviders() []string {
	names := make([]string, 0, len(s.idps))
	for name := range s.idps {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ActiveIdentity returns the name of the signed-in identity provider, or "".
func (s *Service) ActiveIdentity() string { return s.activeIdpName }

// LoginWith signs in using a named identity provider (browser once) and makes
// it active.
func (s *Service) LoginWith(ctx context.Context, name string) error {
	idp, ok := s.idps[name]
	if !ok {
		return fmt.Errorf("service: unknown identity provider %q", name)
	}
	if err := idp.Login(ctx); err != nil {
		return err
	}
	s.activeIdpName = name
	return nil
}

// Login is a convenience for the common single-provider case. With several
// providers configured it returns an error asking the caller to use LoginWith.
func (s *Service) Login(ctx context.Context) error {
	switch len(s.idps) {
	case 0:
		return errors.New("service: no identity provider configured")
	case 1:
		return s.LoginWith(ctx, s.IdentityProviders()[0])
	default:
		return fmt.Errorf("service: multiple identity providers (%v) — use LoginWith", s.IdentityProviders())
	}
}

func (s *Service) IsAuthenticated() bool {
	idp, err := s.activeIDP()
	return err == nil && idp.IsAuthenticated()
}

func (s *Service) Logout(ctx context.Context) error {
	idp, err := s.activeIDP()
	if err != nil {
		return nil
	}
	err = idp.Logout(ctx)
	s.activeIdpName = ""
	return err
}

func (s *Service) activeIDP() (core.IdentityProvider, error) {
	if s.activeIdpName == "" {
		return nil, core.ErrNotAuthenticated
	}
	idp, ok := s.idps[s.activeIdpName]
	if !ok {
		return nil, fmt.Errorf("service: active identity %q not found", s.activeIdpName)
	}
	return idp, nil
}

// Providers lists the registered cloud-provider IDs.
func (s *Service) Providers() []core.ProviderID { return s.registry.IDs() }

// ProviderUsable reports whether a provider can be used with the currently
// active identity. Azure, for instance, is unusable unless the active identity
// is an Entra issuer. Before any identity is chosen it returns true.
func (s *Service) ProviderUsable(id core.ProviderID) bool {
	c, ok := s.registry.Get(id)
	if !ok {
		return false
	}
	con, ok := c.(core.IdentityConstraint)
	if !ok {
		return true
	}
	if s.activeIdpName == "" {
		return true
	}
	idp, err := s.activeIDP()
	if err != nil {
		return false
	}
	return con.AcceptsIssuer(core.IssuerOf(idp))
}

// Connect derives a provider's credentials from the ACTIVE identity (silent).
func (s *Service) Connect(ctx context.Context, id core.ProviderID) error {
	idp, err := s.activeIDP()
	if err != nil {
		return err
	}
	c, ok := s.registry.Get(id)
	if !ok {
		return fmt.Errorf("service: unknown provider %q", id)
	}
	if con, ok := c.(core.IdentityConstraint); ok && !con.AcceptsIssuer(core.IssuerOf(idp)) {
		return fmt.Errorf("service: %q requires a Microsoft Entra identity (active identity %q cannot obtain its credentials)", id, s.activeIdpName)
	}
	return c.Authenticate(ctx, idp)
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

// Init builds the shared Service: one generic OIDC identity provider per
// configured "identity" entry, plus every registered + configured connector.
// warnings lists identity/provider sections skipped due to incomplete config
// (the app still runs with the rest); a non-nil error is fatal (bad config.json
// or zero usable identity providers).
func Init() (svc *Service, warnings []error, err error) {
	secs, err := config.Load()
	if err != nil {
		return nil, nil, err
	}

	idps := make(map[string]core.IdentityProvider)
	names := make([]string, 0, len(secs.Identity))
	for name := range secs.Identity {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		cfg, perr := oidc.ParseConfig(secs.Identity[name])
		if perr != nil {
			warnings = append(warnings, fmt.Errorf("identity %q: %w", name, perr))
			continue
		}
		idps[name] = oidc.New(name, cfg)
	}

	reg, provWarn := core.BuildAll(core.Factories(), secs.Providers)
	warnings = append(warnings, provWarn...)

	if len(idps) == 0 {
		return nil, warnings, fmt.Errorf(`bootstrap: no identity provider configured — fill "identity" in config.json`)
	}
	return NewService(idps, reg), warnings, nil
}
