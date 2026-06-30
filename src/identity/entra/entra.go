// Package entra implements core.IdentityProvider for Microsoft Entra ID using
// the generic OAuth 2.0 Authorization Code + PKCE flow (core/auth).

package entra

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/muhamm-ad/stratus/core"
	"github.com/muhamm-ad/stratus/core/auth"
)

// loginScope yields an id_token (aud = client_id) plus a refresh token.
const loginScope = "openid offline_access"

// refreshWindow is how long before expiry a token is proactively refreshed.
const refreshWindow = 5 * time.Minute

// Provider implements core.IdentityProvider for Entra ID.
type Provider struct {
	cfg         Config
	store       auth.TokenStore
	open        func(string) error
	httpClient  *http.Client
	now         func() time.Time
	authorizeEP string
	tokenEP     string

	mu  sync.Mutex
	tok *auth.TokenResponse
}

// Option customizes a Provider (tests inject fakes).
type Option func(*Provider)

func WithTokenStore(s auth.TokenStore) Option       { return func(p *Provider) { p.store = s } }
func WithBrowserOpener(f func(string) error) Option { return func(p *Provider) { p.open = f } }
func WithHTTPClient(c *http.Client) Option          { return func(p *Provider) { p.httpClient = c } }
func WithClock(f func() time.Time) Option           { return func(p *Provider) { p.now = f } }

// WithEndpoints overrides the Entra authorize/token endpoints (tests).
func WithEndpoints(authorizeEP, tokenEP string) Option {
	return func(p *Provider) { p.authorizeEP = authorizeEP; p.tokenEP = tokenEP }
}

// New builds an Entra identity provider from config.
func New(cfg Config, opts ...Option) *Provider {
	p := &Provider{
		cfg:         cfg,
		store:       auth.NewKeyringStore("entra:" + cfg.TenantID),
		open:        auth.OpenURL,
		httpClient:  http.DefaultClient,
		now:         time.Now,
		authorizeEP: "https://login.microsoftonline.com/" + cfg.TenantID + "/oauth2/v2.0/authorize",
		tokenEP:     "https://login.microsoftonline.com/" + cfg.TenantID + "/oauth2/v2.0/token",
	}
	for _, opt := range opts {
		opt(p)
	}
	if t, err := p.store.Load(); err == nil && t != nil {
		p.tok = t
	}
	return p
}

func (p *Provider) oidc(scope string) auth.OIDCConfig {
	return auth.OIDCConfig{
		ClientID:          p.cfg.ClientID,
		AuthorizeEndpoint: p.authorizeEP,
		TokenEndpoint:     p.tokenEP,
		Scope:             scope,
		HTTPClient:        p.httpClient,
		Now:               p.now,
	}
}

// Login opens the browser once and caches the Entra session.
func (p *Provider) Login(ctx context.Context) error {
	tok, err := auth.LoginAuthCode(ctx, p.oidc(loginScope), p.open)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return core.ErrLoginCancelled
		}
		return err
	}
	p.mu.Lock()
	p.tok = tok
	p.mu.Unlock()
	_ = p.store.Save(tok)
	return nil
}

func (p *Provider) IsAuthenticated() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.tok.Valid(p.now())
}

// IDToken returns the current id_token, refreshing it if it is expiring soon.
func (p *Provider) IDToken(ctx context.Context) (string, error) {
	p.mu.Lock()
	tok := p.tok
	p.mu.Unlock()
	if tok == nil || tok.IDToken == "" {
		return "", core.ErrNotAuthenticated
	}
	if tok.ExpiringSoon(p.now(), refreshWindow) {
		nt, err := auth.RefreshToken(ctx, p.oidc(loginScope), tok.RefreshToken)
		if err != nil {
			return "", core.ErrTokenExpired
		}
		if nt.RefreshToken == "" {
			nt.RefreshToken = tok.RefreshToken
		}
		p.mu.Lock()
		p.tok = nt
		p.mu.Unlock()
		_ = p.store.Save(nt)
		return nt.IDToken, nil
	}
	return tok.IDToken, nil
}

// AccessToken silently acquires an access token for an arbitrary scope using
// the refresh token (e.g. the ARM scope for Azure). No browser.
func (p *Provider) AccessToken(ctx context.Context, scope string) (string, error) {
	p.mu.Lock()
	var rt string
	if p.tok != nil {
		rt = p.tok.RefreshToken
	}
	p.mu.Unlock()
	if rt == "" {
		return "", core.ErrNotAuthenticated
	}
	resp, err := auth.RefreshToken(ctx, p.oidc(scope), rt)
	if err != nil {
		return "", core.ErrTokenExpired
	}
	// Entra rotates refresh tokens; keep the newest.
	if resp.RefreshToken != "" {
		p.mu.Lock()
		if p.tok != nil {
			p.tok.RefreshToken = resp.RefreshToken
			_ = p.store.Save(p.tok)
		}
		p.mu.Unlock()
	}
	return resp.AccessToken, nil
}

func (p *Provider) Logout(ctx context.Context) error {
	p.mu.Lock()
	p.tok = nil
	p.mu.Unlock()
	return p.store.Clear()
}

var _ core.IdentityProvider = (*Provider)(nil)
