package gcp

import (
	"context"
	"fmt"

	"golang.org/x/oauth2/google/externalaccount"

	"github.com/muhamm-ad/stratus/core"
)

// subjectTokenSupplier feeds the OIDC id_token to Google's STS token exchange
// (RFC 8693) instead of reading it from a file.
type subjectTokenSupplier struct {
	ctx context.Context
	idp core.IdentityProvider
}

func (s subjectTokenSupplier) SubjectToken(ctx context.Context, _ externalaccount.SupplierOptions) (string, error) {
	return s.idp.IDToken(ctx)
}

type Provider struct {
	cfg   Config
	token string
	ok    bool
}

func New(cfg Config) *Provider { return &Provider{cfg: cfg} }

func (p *Provider) ID() core.ProviderID { return core.ProviderGCP }

func (p *Provider) Authenticate(ctx context.Context, idp core.IdentityProvider) error {
	ts, err := externalaccount.NewTokenSource(ctx, externalaccount.Config{
		Audience:                 p.cfg.WorkforceAudience,
		SubjectTokenType:         "urn:ietf:params:oauth:token-type:jwt",
		TokenURL:                 "https://sts.googleapis.com/v1/token",
		Scopes:                   []string{p.cfg.Scope},
		SubjectTokenSupplier:     subjectTokenSupplier{ctx: ctx, idp: idp},
		WorkforcePoolUserProject: p.cfg.WorkforcePoolUserProject,
	})
	if err != nil {
		return fmt.Errorf("%w: gcp externalaccount: %v", core.ErrExchange, err)
	}
	tok, err := ts.Token()
	if err != nil {
		return fmt.Errorf("%w: gcp STS token-exchange: %v", core.ErrExchange, err)
	}
	p.token, p.ok = tok.AccessToken, true
	return nil
}

func (p *Provider) IsAuthenticated() bool { return p.ok }
func (p *Provider) AccessToken() string   { return p.token }

func (p *Provider) ListInstances(ctx context.Context, account string) ([]core.Instance, error) {
	return nil, core.ErrNotImplemented // Phase 3: Compute aggregatedList
}
func (p *Provider) Connect(ctx context.Context, req core.ConnectRequest) (core.Session, error) {
	return nil, core.ErrNotImplemented // Phase 4
}
func (p *Provider) Logout(ctx context.Context) error { p.token, p.ok = "", false; return nil }

func (p *Provider) GetAccount() (string, error) {
	return p.cfg.WorkforcePoolUserProject, nil
}