// Package gcp implements core.ProviderConnector for GCP: it exchanges
// the Entra id_token for a short-lived GCP access token via the Security Token
// Service (OAuth 2.0 Token Exchange, RFC 8693).
package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/muhamm-ad/stratus/core"
)

// defaultSTSEndpoint is the Google STS token-exchange endpoint.
const defaultSTSEndpoint = "https://sts.googleapis.com/v1/token"

// Provider implements core.ProviderConnector for GCP.
type Provider struct {
	cfg         Config
	httpClient  *http.Client
	stsEndpoint string
	now         func() time.Time

	mu        sync.Mutex
	token     string
	expiresAt time.Time
}

type Option func(*Provider)

func WithHTTPClient(c *http.Client) Option { return func(p *Provider) { p.httpClient = c } }
func WithSTSEndpoint(url string) Option    { return func(p *Provider) { p.stsEndpoint = url } }
func WithClock(f func() time.Time) Option  { return func(p *Provider) { p.now = f } }

func New(cfg Config, opts ...Option) *Provider {
	p := &Provider{cfg: cfg, httpClient: http.DefaultClient, stsEndpoint: defaultSTSEndpoint, now: time.Now}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func (p *Provider) ID() core.ProviderID { return core.ProviderGCP }

func (p *Provider) IsAuthenticated() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.token != "" && p.now().Before(p.expiresAt)
}

// Authenticate exchanges the Entra id_token for a GCP access token.
func (p *Provider) Authenticate(ctx context.Context, idp core.IdentityProvider) error {
	idToken, err := idp.IDToken(ctx)
	if err != nil {
		return fmt.Errorf("gcp: %w", err)
	}
	form := url.Values{
		"grant_type":           {"urn:ietf:params:oauth:grant-type:token-exchange"},
		"audience":             {p.cfg.WorkforceAudience},
		"requested_token_type": {"urn:ietf:params:oauth:token-type:access_token"},
		"subject_token_type":   {"urn:ietf:params:oauth:token-type:jwt"},
		"subject_token":        {idToken},
		"scope":                {p.cfg.Scope},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.stsEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: gcp sts request: %v", core.ErrExchange, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: gcp sts returned %d: %s", core.ErrExchange, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return fmt.Errorf("%w: decode gcp sts response: %v", core.ErrExchange, err)
	}
	p.mu.Lock()
	p.token = out.AccessToken
	p.expiresAt = p.now().Add(time.Duration(out.ExpiresIn) * time.Second)
	p.mu.Unlock()
	return nil
}

// AccessToken returns the held GCP access token (used later by the connection layer).
func (p *Provider) AccessToken() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.token
}

func (p *Provider) Logout(ctx context.Context) error {
	p.mu.Lock()
	p.token = ""
	p.expiresAt = time.Time{}
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
