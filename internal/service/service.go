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
func (s *Service) GetActiveIdentityProviderID() core.IdentityProviderID { return s.activeIdentityID }

func (s *Service) GetActiveIdentityProvider() core.IdentityProvider {
	return s.idps[s.activeIdentityID]
}

// UsesIdentityUsesDeviceFlow reports whether the named IdP is configured for RFC
// 8628 device flow instead of the default browser+loopback flow.
func (s *Service) UsesIdentityUsesDeviceFlow(id core.IdentityProviderID) bool {
	idp, ok := s.idps[id]
	if !ok {
		return false
	}
	if o, ok := idp.(*oidc.OIDCIdentityProvider); ok {
		return o.UsesDeviceFlow()
	}
	return false
}

// LoginWith signs in using an identity provider (browser once), makes it
// active, then silently authenticates every registered cloud provider.
// onCode is called only if the device flow is used (may be nil).
// A non-nil error means identity login failed. Per-cloud failures are
// returned in the map without failing the overall login.
func (s *Service) LoginWith(ctx context.Context, id core.IdentityProviderID, onCode func(core.DeviceCode)) (core.IdentityProvider, error, map[core.CloudProviderID]error) {
	idp, ok := s.idps[id]
	if !ok {
		return nil, fmt.Errorf("service: unknown identity provider %q", id), nil
	}
	s.activeIdentityID = ""
	if err := idp.Login(ctx, onCode); err != nil {
		return nil, err, nil
	}
	s.activeIdentityID = id

	cpErrors := make(map[core.CloudProviderID]error)
	for _, cpID := range s.GetCloudProvidersIDs() {
		if err := s.authenticateCloudProvider(ctx, cpID); err != nil {
			cpErrors[cpID] = err
		}
	}
	return idp, nil, cpErrors
}

// Login is a convenience for the common single-provider case. With several
// providers configured it returns an error asking the caller to use LoginWith.
func (s *Service) Login(ctx context.Context, onCode func(core.DeviceCode)) (core.IdentityProvider, error, map[core.CloudProviderID]error) {
	switch len(s.idps) {
	case 0:
		return nil, errors.New("service: no identity provider configured"), nil
	case 1:
		return s.LoginWith(ctx, s.IdentityProvidersIDs()[0], onCode)
	default:
		return nil, fmt.Errorf("service: multiple identity providers (%v) — use LoginWith", s.IdentityProviders()), nil
	}
}

func (s *Service) IsAuthenticated() bool {
	idp, err := s.activeIdentityProvider()
	if err != nil {
		return false
	}
	return idp.IsAuthenticated()
}

func (s *Service) Logout(ctx context.Context) error {
	idp, err := s.activeIdentityProvider()
	if err != nil {
		return nil
	}
	err = idp.Logout(ctx)
	s.activeIdentityID = ""
	return err
}

func (s *Service) activeIdentityProvider() (core.IdentityProvider, error) {
	if s.activeIdentityID == "" {
		return nil, core.ErrNotAuthenticated
	}
	return s.idps[s.activeIdentityID], nil
}

// GetCloudProvidersIDs lists the registered cloud-provider IDs.
func (s *Service) GetCloudProvidersIDs() []core.CloudProviderID {
	return s.registry.IDs()
}

func (s *Service) GetCloudProviders() map[core.CloudProviderID]core.CloudProvider {
	return s.registry.GetAll()
}

func (s *Service) GetCloudProviderStatus(id core.CloudProviderID) core.CloudProviderStatus {
	cp, ok := s.registry.Get(id)
	if !ok {
		return core.CloudProviderStatusUnknown
	}
	return cp.GetStatus()
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
	idp, err := s.activeIdentityProvider()
	if err != nil {
		return true
	}
	return con.AcceptsIdentityIssuer(core.IssuerOf(idp))
}

// AuthenticateCloudProvider derives a provider's credentials from the ACTIVE identity (silent).
func (s *Service) authenticateCloudProvider(ctx context.Context, id core.CloudProviderID) error {
	idp, err := s.activeIdentityProvider()
	if err != nil {
		return err
	}
	cp, ok := s.registry.Get(id)
	if !ok {
		return fmt.Errorf("service: unknown provider %q", id)
	}
	if con, ok := cp.(core.CloudProviderConstraint); ok && !con.AcceptsIdentityIssuer(core.IssuerOf(idp)) {
		return fmt.Errorf("service: %q requires a Microsoft Entra identity (active identity %q cannot obtain its credentials)", id, s.activeIdentityID)
	}
	if cp.IsAuthenticated() && cp.GetStatus() != core.CloudProviderStatusError {
		return nil
	}
	return cp.Authenticate(ctx, idp)
}

func (s *Service) ReconnectProvider(ctx context.Context, id core.CloudProviderID) error {
	return s.authenticateCloudProvider(ctx, id)
}

// ListVMs returns connectable VMs for one account on a provider. (Phase 3.)
func (s *Service) ListVMs(ctx context.Context, id core.CloudProviderID) ([]core.VM, error) {
	cp, ok := s.registry.Get(id)
	if !ok {
		return nil, fmt.Errorf("service: unknown account for provider %q", id)
	}
	return cp.ListVMs(ctx)
}

// OpenSession opens an interactive session to one instance. (Phase 4.)
func (s *Service) OpenSession(ctx context.Context, id core.CloudProviderID, req core.ConnectRequest) (core.Session, error) {
	c, ok := s.registry.Get(id)
	if !ok {
		return nil, fmt.Errorf("service: unknown provider %q", id)
	}
	return c.Connect(ctx, req)
}
