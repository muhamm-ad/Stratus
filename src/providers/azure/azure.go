package azure

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/muhamm-ad/stratus/core"
)

// Provider acquires an ARM-scoped token from the ACTIVE identity's refresh
// token (delegated to x/oauth2 inside the auth.Session). This preserves the
// single sign-on: no second browser, no separate Azure login. ARM only accepts
// Entra-issued tokens, hence AcceptsIssuer (core.IdentityConstraint).
type Provider struct {
	cfg Config

	mu       sync.Mutex
	armToken string
	ok       bool
}

func New(cfg Config) *Provider {
	if cfg.ARMScope == "" {
		cfg.ARMScope = DefaultARMScope
	}
	return &Provider{cfg: cfg}
}

func (p *Provider) ID() core.ProviderID { return core.ProviderAzure }

func (p *Provider) Authenticate(ctx context.Context, idp core.IdentityProvider) error {
	tok, err := idp.AccessToken(ctx, p.cfg.ARMScope)
	if err != nil {
		return fmt.Errorf("%w: azure ARM token: %v", core.ErrExchange, err)
	}
	p.mu.Lock()
	p.armToken, p.ok = tok, true
	p.mu.Unlock()
	return nil
}

func (p *Provider) IsAuthenticated() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.ok
}

func (p *Provider) AccessToken() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.armToken
}

// AcceptsIssuer implements core.IdentityConstraint: ARM requires an Entra token.
func (p *Provider) AcceptsIssuer(issuer string) bool {
	return strings.Contains(issuer, "login.microsoftonline.com") ||
		strings.Contains(issuer, "sts.windows.net")
}

func (p *Provider) ListInstances(ctx context.Context, account string) ([]core.Instance, error) {
	return nil, core.ErrNotImplemented // Phase 3: ARM VM list
}
func (p *Provider) Connect(ctx context.Context, req core.ConnectRequest) (core.Session, error) {
	return nil, core.ErrNotImplemented // Phase 4
}
func (p *Provider) Logout(ctx context.Context) error {
	p.mu.Lock()
	p.armToken, p.ok = "", false
	p.mu.Unlock()
	return nil
}
