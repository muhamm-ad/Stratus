package aws

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ssooidc"

	"github.com/muhamm-ad/stratus/core"
)

// refreshWindow is how long before expiry a token is proactively refreshed.
const refreshWindow = 5 * time.Minute

// Provider implements core.ProviderConnector for AWS IAM Identity Center.
type Provider struct {
	cfg          Config
	oidc         OIDCClient
	store        TokenStore
	openBrowser  func(string) error
	onDeviceCode func(verificationURI, userCode string)
	now          func() time.Time

	mu    sync.Mutex
	token *Token
}

// Option customizes a Provider. Tests use these to inject fakes.
type Option func(*Provider)

// WithTokenStore overrides the default OS-keychain token store.
func WithTokenStore(s TokenStore) Option { return func(p *Provider) { p.store = s } }

// WithBrowserOpener overrides how authorization URLs are opened.
func WithBrowserOpener(f func(string) error) Option { return func(p *Provider) { p.openBrowser = f } }

// WithDeviceCodeHandler registers a callback that receives the verification URI
// and user code during the device flow (and the URL if the browser fails to
// open during the PKCE flow).
func WithDeviceCodeHandler(f func(uri, code string)) Option {
	return func(p *Provider) { p.onDeviceCode = f }
}

// WithOIDCClient injects a custom OIDC client (used in tests).
func WithOIDCClient(c OIDCClient) Option { return func(p *Provider) { p.oidc = c } }

// WithClock injects a deterministic time source (used in tests).
func WithClock(f func() time.Time) Option { return func(p *Provider) { p.now = f } }

// NewProvider builds an AWS connector from an already-parsed configuration.
// The caller (main) obtains cfg via aws.ParseConfig(section), keeping config
// loading out of the connector and making the provider trivially testable.
func NewProvider(cfg Config, opts ...Option) (*Provider, error) {
	p := &Provider{
		cfg:         cfg,
		oidc:        ssooidc.New(ssooidc.Options{Region: cfg.SSORegion}),
		store:       newKeyringStore(cfg.SSOStartURL),
		openBrowser: openURL,
		now:         time.Now,
	}
	for _, opt := range opts {
		opt(p)
	}

	// Best-effort restore of a cached session; ignore storage errors so a
	// missing/locked keychain never blocks startup.
	if t, err := p.store.Load(); err == nil && t != nil {
		p.token = t
	}
	return p, nil
}

func (p *Provider) ID() core.ProviderID { return core.ProviderAWS }

func (p *Provider) IsAuthenticated() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.token.Valid(p.now())
}

// Login runs the interactive auth flow and persists the resulting session.
func (p *Provider) Login(ctx context.Context) error {
	tok, err := p.authenticate(ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return core.ErrLoginCancelled
		}
		return err
	}

	p.mu.Lock()
	p.token = tok
	p.mu.Unlock()

	_ = p.store.Save(tok) // best-effort persistence
	return nil
}

// validToken returns a usable access token, refreshing it if it is expiring
// soon. Used by Phase 3 operations (ListAccounts, etc.).
func (p *Provider) validToken(ctx context.Context) (*Token, error) {
	p.mu.Lock()
	tok := p.token
	p.mu.Unlock()

	if tok == nil || tok.AccessToken == "" {
		return nil, core.ErrNotAuthenticated
	}
	if tok.Valid(p.now()) && !tok.expiringSoon(p.now(), refreshWindow) {
		return tok, nil
	}

	nt, err := p.refresh(ctx, tok)
	if err != nil {
		return nil, err
	}
	p.mu.Lock()
	p.token = nt
	p.mu.Unlock()
	_ = p.store.Save(nt)
	return nt, nil
}

func (p *Provider) Logout(ctx context.Context) error {
	p.mu.Lock()
	p.token = nil
	p.mu.Unlock()
	return p.store.Clear()
}

// --- Phase 3 / 4: implemented in later milestones -------------------------

func (p *Provider) ListAccounts(ctx context.Context) ([]core.Account, error) {
	return nil, core.ErrNotImplemented
}

func (p *Provider) ListInstances(ctx context.Context, accountID string) ([]core.Instance, error) {
	return nil, core.ErrNotImplemented
}

func (p *Provider) Connect(ctx context.Context, req core.ConnectRequest) (core.Session, error) {
	return nil, core.ErrNotImplemented
}

// Compile-time assertion that Provider satisfies the core contract.
var _ core.ProviderConnector = (*Provider)(nil)
