package oidc

import (
	"context"
	"sync"

	"github.com/muhamm-ad/stratus/core"
	"github.com/muhamm-ad/stratus/core/auth"
)

// Identity is the generic OIDC identity provider. It lazily builds an
// auth.Client (discovery via go-oidc when an issuer is set) and delegates the
// login/refresh lifecycle to auth.Session.
type Identity struct {
	name string
	cfg  Config

	mu   sync.Mutex
	sess *auth.Session
}

func New(name string, cfg Config) *Identity { return &Identity{name: name, cfg: cfg} }

var _ core.IdentityProvider = (*Identity)(nil)

func (p *Identity) ensure(ctx context.Context) (*auth.Session, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.sess != nil {
		return p.sess, nil
	}
	var (
		client *auth.Client
		err    error
	)
	if p.cfg.Issuer != "" {
		client, err = auth.NewClient(ctx, p.cfg.Issuer, p.cfg.ClientID, p.cfg.scopes())
		if err != nil {
			return nil, err
		}
	} else {
		client = auth.NewClientManual(p.cfg.ClientID, p.cfg.AuthorizeEndpoint, p.cfg.TokenEndpoint, p.cfg.scopes())
	}
	store := auth.Store{Service: "stratus", Key: "oidc:" + p.name}
	p.sess = auth.NewSession(client, store, p.cfg.UseDeviceFlow)
	return p.sess, nil
}

func (p *Identity) Login(ctx context.Context, onCode func(core.DeviceCode)) error {
	s, err := p.ensure(ctx)
	if err != nil {
		return err
	}
	return s.Login(ctx, onCode)
}

func (p *Identity) IsAuthenticated() bool {
	p.mu.Lock()
	s := p.sess
	p.mu.Unlock()
	if s == nil {
		// Peek the keychain without building a client (no network).
		t, _, err := (auth.Store{Service: "stratus", Key: "oidc:" + p.name}).Load()
		return err == nil && t.Valid()
	}
	return s.IsAuthenticated()
}

func (p *Identity) IDToken(ctx context.Context) (string, error) {
	s, err := p.ensure(ctx)
	if err != nil {
		return "", err
	}
	return s.IDToken(ctx)
}

func (p *Identity) AccessToken(ctx context.Context, scopes ...string) (string, error) {
	s, err := p.ensure(ctx)
	if err != nil {
		return "", err
	}
	return s.AccessToken(ctx, scopes...)
}

func (p *Identity) Logout(ctx context.Context) error {
	s, err := p.ensure(ctx)
	if err != nil {
		return (auth.Store{Service: "stratus", Key: "oidc:" + p.name}).Clear()
	}
	return s.Logout(ctx)
}

// Issuer reports the configured OIDC issuer (empty when the provider is
// configured with explicit endpoints instead of an issuer). Used by the service
// to evaluate connector identity constraints (e.g. Azure requires Entra).
func (p *Identity) Issuer() string { return p.cfg.Issuer }
