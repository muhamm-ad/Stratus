package azure

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/muhamm-ad/stratus/internal/core"
)

// AzureProvider acquires an ARM-scoped token from the ACTIVE identity's refresh
// token (delegated to x/oauth2 inside the auth.Session). This preserves the
// single sign-on: no second browser, no separate Azure login. ARM only accepts
// Entra-issued tokens, hence AcceptsIssuer (core.IdentityConstraint).
type AzureProvider struct {
	cfg Config

	mu       sync.Mutex
	armToken string
	ok       bool
}

const ProviderID core.CloudProviderID = "azure"

func NewAzureProvider(cfg Config) (*AzureProvider, error) {
	if cfg.ARMScope == "" {
		cfg.ARMScope = DefaultARMScope
	}
	return &AzureProvider{cfg: cfg}, nil
}

func (p *AzureProvider) ID() core.CloudProviderID { return ProviderID }

func (p *AzureProvider) Authenticate(ctx context.Context, idp core.IdentityProvider) error {
	tok, err := idp.AccessToken(ctx, p.cfg.ARMScope)
	if err != nil {
		return fmt.Errorf("%w: azure ARM token: %v", core.ErrExchange, err)
	}
	p.mu.Lock()
	p.armToken, p.ok = tok, true
	p.mu.Unlock()
	return nil
}

func (p *AzureProvider) IsAuthenticated() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.ok
}

func (p *AzureProvider) AccessToken() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.armToken
}

// AcceptsIssuer implements core.IdentityConstraint: ARM requires an Entra token.
func (p *AzureProvider) AcceptsIssuer(issuer string) bool {
	return strings.Contains(issuer, "login.microsoftonline.com") ||
		strings.Contains(issuer, "sts.windows.net")
}

func (p *AzureProvider) ListVMs(ctx context.Context) ([]core.VM, error) {
	return nil, core.ErrNotImplemented // Phase 3: ARM VM list
}
func (p *AzureProvider) Connect(ctx context.Context, req core.ConnectRequest) (core.Session, error) {
	return nil, core.ErrNotImplemented // Phase 4
}
func (p *AzureProvider) Logout(ctx context.Context) error {
	p.mu.Lock()
	p.armToken, p.ok = "", false
	p.mu.Unlock()
	return nil
}

func (p *AzureProvider) getAccount() (core.Account, error) {
	return core.Account{
		ID:    p.cfg.SubscriptionID,
		Name:  p.cfg.SubscriptionID,
		Roles: []string{p.cfg.ARMScope},
	}, nil
}
