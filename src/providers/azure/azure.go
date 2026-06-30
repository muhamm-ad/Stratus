// Package azure implements core.ProviderConnector for Azure. Azure is
// natively Entra, so we acquire an ARM-scoped access token silently from the
// Entra identity (via its refresh token).
package azure

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/muhamm-ad/stratus/core"
)

// Provider implements core.ProviderConnector for Azure.
type Provider struct {
	cfg Config
	now func() time.Time

	mu    sync.Mutex
	token string
}

type Option func(*Provider)

func WithClock(f func() time.Time) Option { return func(p *Provider) { p.now = f } }

func New(cfg Config, opts ...Option) *Provider {
	if cfg.ARMScope == "" {
		cfg.ARMScope = DefaultARMScope
	}
	p := &Provider{cfg: cfg, now: time.Now}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func (p *Provider) ID() core.ProviderID { return core.ProviderAzure }

func (p *Provider) IsAuthenticated() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.token != ""
}

// Authenticate acquires an ARM-scoped token from the Entra identity, silently.
func (p *Provider) Authenticate(ctx context.Context, idp core.IdentityProvider) error {
	armToken, err := idp.AccessToken(ctx, p.cfg.ARMScope)
	if err != nil {
		return fmt.Errorf("%w: azure acquire ARM token: %v", core.ErrExchange, err)
	}
	p.mu.Lock()
	p.token = armToken
	p.mu.Unlock()
	return nil
}

// AccessToken returns the held ARM token (used later by the connection layer).
func (p *Provider) AccessToken() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.token
}

func (p *Provider) Logout(ctx context.Context) error {
	p.mu.Lock()
	p.token = ""
	p.mu.Unlock()
	return nil
}

func (p *Provider) ListInstances(ctx context.Context, accountID string) ([]core.Instance, error) {
	return nil, core.ErrNotImplemented
}
func (p *Provider) Connect(ctx context.Context, req core.ConnectRequest) (core.Session, error) {
	return nil, core.ErrNotImplemented
}

var _ core.ProviderConnector = (*Provider)(nil)
