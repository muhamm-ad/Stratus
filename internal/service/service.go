// Package service is the shared application layer of Stratus. Both user
// interfaces — the Wails/React desktop UI and the CLI/TUI — call into it;
// neither duplicates orchestration. Init() (formerly the bootstrap package) is
// the single assembly point; the Service methods themselves stay provider-
// agnostic and act only through core interfaces.
package service

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/muhamm-ad/stratus/internal/core"
	"github.com/muhamm-ad/stratus/internal/provider/identity/oidc"

	// Register the built-in cloud providers (their init() calls RegisterProvider).
	_ "github.com/muhamm-ad/stratus/internal/provider/cloud/all"
)

// Service exposes the high-level operations both UIs need. It holds several
// configured identity providers (all OIDC) and the cloud connectors. Exactly
// one identity provider is "active" at a time — the one the user signed in with.
type Service struct {
	idps             map[core.IdentityProviderID]core.IdentityProvider
	activeIdentityID core.IdentityProviderID // key into idps; "" means none chosen
	registry         *core.Registry
}

// NewService builds a Service from the configured identity providers and the
// connector registry. When exactly one identity provider is configured it
// becomes active automatically (no choice to make).
func NewService(idps map[core.IdentityProviderID]core.IdentityProvider, registry *core.Registry) *Service {
	s := &Service{idps: idps, registry: registry}
	if len(idps) == 1 {
		for id := range idps {
			s.activeIdentityID = id
			break
		}
	}
	return s
}

// IdentityProviders lists the configured identity-provider names (e.g. "entra",
// "okta") for the single-sign-on choice, sorted.
func (s *Service) IdentityProvidersIDs() []core.IdentityProviderID {
	ids := make([]core.IdentityProviderID, 0, len(s.idps))
	for id := range s.idps {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func (s *Service) IdentityProviders() map[core.IdentityProviderID]core.IdentityProvider {
	return s.idps
}

// ActiveIdentity returns the name of the signed-in identity provider, or "".
func (s *Service) ActiveIdentityProviderID() core.IdentityProviderID { return s.activeIdentityID }

func (s *Service) ActiveIdentityProvider() core.IdentityProvider {
	return s.idps[s.activeIdentityID]
}

// IdentityUsesDeviceFlow reports whether the named IdP is configured for RFC
// 8628 device flow instead of the default browser+loopback flow.
func (s *Service) IdentityUsesDeviceFlow(id core.IdentityProviderID) bool {
	idp, ok := s.idps[id]
	if !ok {
		return false
	}
	if o, ok := idp.(*oidc.OIDCIdentityProvider); ok {
		return o.UsesDeviceFlow()
	}
	return false
}

// LoginWith signs in using an identity provider (browser once) and makes
// it active. onCode is called only if the device flow is used (may be nil).
func (s *Service) LoginWith(ctx context.Context, id core.IdentityProviderID, onCode func(core.DeviceCode)) (core.IdentityProvider, error) {
	idp, ok := s.idps[id]
	if !ok {
		return nil, fmt.Errorf("service: unknown identity provider %q", id)
	}
	if err := idp.Login(ctx, onCode); err != nil {
		return nil, err
	}
	s.activeIdentityID = id
	return idp, nil
}

// Login is a convenience for the common single-provider case. With several
// providers configured it returns an error asking the caller to use LoginWith.
func (s *Service) Login(ctx context.Context, onCode func(core.DeviceCode)) (core.IdentityProvider, error) {
	switch len(s.idps) {
	case 0:
		return nil, errors.New("service: no identity provider configured")
	case 1:
		return s.LoginWith(ctx, s.IdentityProvidersIDs()[0], onCode)
	default:
		return nil, fmt.Errorf("service: multiple identity providers (%v) — use LoginWith", s.IdentityProviders())
	}
}

func (s *Service) IsAuthenticated() bool {
	idp, ok := s.activeIdentityProvider()
	return ok && idp.IsAuthenticated()
}

func (s *Service) Logout(ctx context.Context) error {
	idp, ok := s.activeIdentityProvider()
	if !ok {
		return nil
	}
	err := idp.Logout(ctx)
	s.activeIdentityID = ""
	return err
}

func (s *Service) activeIdentityProvider() (core.IdentityProvider, bool) {
	if s.activeIdentityID == "" {
		return nil, false
	}
	idp, ok := s.idps[s.activeIdentityID]
	return idp, ok
}

func (s *Service) requireActive() (core.IdentityProvider, error) {
	idp, ok := s.activeIdentityProvider()
	if !ok {
		return nil, core.ErrNotAuthenticated
	}
	return idp, nil
}

// CloudProviders lists the registered cloud-provider IDs.
func (s *Service) CloudProviders() []core.CloudProviderID { return s.registry.IDs() }

// Accounts returns configured account/subscription/project IDs per provider.
func (s *Service) Accounts() map[core.CloudProviderID]string {
	out := make(map[core.CloudProviderID]string)
	for _, id := range s.registry.IDs() {
		c, ok := s.registry.Get(id)
		if !ok {
			continue
		}
		account, err := c.GetAccount()
		if err != nil {
			continue
		}
		out[id] = account
	}
	return out
}

// ProviderUsable reports whether a provider can be used with the currently
// active identity. Azure, for instance, is unusable unless the active identity
// is an Entra issuer. Before any identity is chosen it returns true.
func (s *Service) ProviderUsable(id core.CloudProviderID) bool {
	c, ok := s.registry.Get(id)
	if !ok {
		return false
	}
	con, ok := c.(core.CloudProviderConstraint)
	if !ok {
		return true
	}
	idp, ok := s.activeIdentityProvider()
	if !ok {
		return true
	}
	return con.AcceptsIdentityIssuer(core.IssuerOf(idp))
}

// Connect derives a provider's credentials from the ACTIVE identity (silent).
func (s *Service) Connect(ctx context.Context, id core.CloudProviderID) error {
	idp, err := s.requireActive()
	if err != nil {
		return err
	}
	c, ok := s.registry.Get(id)
	if !ok {
		return fmt.Errorf("service: unknown provider %q", id)
	}
	if con, ok := c.(core.CloudProviderConstraint); ok && !con.AcceptsIdentityIssuer(core.IssuerOf(idp)) {
		return fmt.Errorf("service: %q requires a Microsoft Entra identity (active identity %q cannot obtain its credentials)", id, s.activeIdentityID)
	}
	return c.Authenticate(ctx, idp)
}

// ListInstances returns connectable VMs for one account on a provider. (Phase 3.)
func (s *Service) ListInstances(ctx context.Context, id core.CloudProviderID, account string) ([]core.Instance, error) {
	cp, ok := s.registry.Get(id)
	if !ok {
		return nil, fmt.Errorf("service: unknown provider %q", id)
	}
	return cp.ListInstances(ctx, account)
}

// OpenSession opens an interactive session to one instance. (Phase 4.)
func (s *Service) OpenSession(ctx context.Context, id core.CloudProviderID, req core.ConnectRequest) (core.Session, error) {
	c, ok := s.registry.Get(id)
	if !ok {
		return nil, fmt.Errorf("service: unknown provider %q", id)
	}
	return c.Connect(ctx, req)
}

