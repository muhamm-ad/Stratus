package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/muhamm-ad/stratus/core"
	"github.com/muhamm-ad/stratus/core/auth"
)

// Provider is a generic OIDC identity provider. It lazily resolves its
// endpoints (via discovery when only an issuer is configured), then delegates
// the login/refresh lifecycle to a reusable auth.Session. One instance per
// configured provider (Entra, Okta, …).
type Provider struct {
	name string
	cfg  Config

	store auth.TokenStore
	open  func(string) error
	hc    *http.Client
	now   func() time.Time

	mu   sync.Mutex
	sess *auth.Session
}

// Option configures a Provider (defaults are production-ready; options are for
// tests and custom stores).
type Option func(*Provider)

func WithTokenStore(s auth.TokenStore) Option       { return func(p *Provider) { p.store = s } }
func WithBrowserOpener(f func(string) error) Option { return func(p *Provider) { p.open = f } }
func WithHTTPClient(c *http.Client) Option          { return func(p *Provider) { p.hc = c } }
func WithClock(f func() time.Time) Option           { return func(p *Provider) { p.now = f } }

// New builds a generic OIDC provider. name is the config key (e.g. "entra"),
// used to namespace the persisted token.
func New(name string, cfg Config, opts ...Option) *Provider {
	p := &Provider{
		name: name,
		cfg:  cfg,
		open: auth.OpenURL,
		hc:   http.DefaultClient,
		now:  time.Now,
	}
	for _, o := range opts {
		o(p)
	}
	if p.store == nil {
		p.store = auth.NewKeyringStore("oidc:" + name)
	}
	return p
}

var _ core.IdentityProvider = (*Provider)(nil)

// Name returns the configured provider key.
func (p *Provider) Name() string { return p.name }

// discover fetches the provider's OIDC discovery document and returns the
// authorize and token endpoints.
func discover(ctx context.Context, hc *http.Client, issuer string) (authorizeEP, tokenEP string, err error) {
	url := strings.TrimRight(issuer, "/") + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", "", fmt.Errorf("oidc: discovery request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := hc.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("oidc: discovery fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", "", fmt.Errorf("oidc: discovery %s: status %d: %s", url, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var d struct {
		AuthorizationEndpoint string `json:"authorization_endpoint"`
		TokenEndpoint         string `json:"token_endpoint"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&d); err != nil {
		return "", "", fmt.Errorf("oidc: decode discovery %s: %w", url, err)
	}
	if d.AuthorizationEndpoint == "" || d.TokenEndpoint == "" {
		return "", "", fmt.Errorf("oidc: discovery %s missing endpoints", url)
	}
	return d.AuthorizationEndpoint, d.TokenEndpoint, nil
}

// ensureSession resolves endpoints (discovery if needed) and builds the Session
// exactly once.
func (p *Provider) ensureSession(ctx context.Context) (*auth.Session, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.sess != nil {
		return p.sess, nil
	}
	authEP, tokEP := p.cfg.AuthorizeEndpoint, p.cfg.TokenEndpoint
	if authEP == "" || tokEP == "" {
		a, t, err := discover(ctx, p.hc, p.cfg.Issuer)
		if err != nil {
			return nil, err
		}
		authEP, tokEP = a, t
	}
	oc := auth.OIDCConfig{
		ClientID:          p.cfg.ClientID,
		AuthorizeEndpoint: authEP,
		TokenEndpoint:     tokEP,
		Scope:             p.cfg.scopeString(),
		HTTPClient:        p.hc,
		Now:               p.now,
	}
	p.sess = auth.NewSession(oc, p.store, p.open)
	return p.sess, nil
}

func (p *Provider) Login(ctx context.Context) error {
	s, err := p.ensureSession(ctx)
	if err != nil {
		return err
	}
	return s.Login(ctx)
}

// IsAuthenticated checks the (possibly persisted) session WITHOUT a network
// call, so it is safe at startup before any endpoint discovery.
func (p *Provider) IsAuthenticated() bool {
	p.mu.Lock()
	s := p.sess
	p.mu.Unlock()
	if s != nil {
		return s.IsAuthenticated()
	}
	t, err := p.store.Load()
	return err == nil && t.Valid(p.now())
}

func (p *Provider) IDToken(ctx context.Context) (string, error) {
	s, err := p.ensureSession(ctx)
	if err != nil {
		return "", err
	}
	return s.IDToken(ctx)
}

func (p *Provider) AccessToken(ctx context.Context, scope string) (string, error) {
	s, err := p.ensureSession(ctx)
	if err != nil {
		return "", err
	}
	return s.AccessToken(ctx, scope)
}

func (p *Provider) Logout(ctx context.Context) error {
	s, err := p.ensureSession(ctx)
	if err != nil {
		return p.store.Clear() // endpoints unresolved: still clear local session
	}
	return s.Logout(ctx)
}

// Issuer reports the configured OIDC issuer (empty when the provider is
// configured with explicit endpoints instead of an issuer). Used by the service
// to evaluate connector identity constraints (e.g. Azure requires Entra).
func (p *Provider) Issuer() string { return p.cfg.Issuer }
