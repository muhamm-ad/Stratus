package gcp

import (
	"context"
	"fmt"

	"golang.org/x/oauth2/google/externalaccount"

	"github.com/muhamm-ad/stratus/internal/core"
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

const ProviderID core.CloudProviderID = "gcp"

type GCPProvider struct {
	cfg   Config
	token string
	ok    bool
}

func NewGCPProvider(cfg Config) (*GCPProvider, error) { return &GCPProvider{cfg: cfg}, nil }

func (p *GCPProvider) ID() core.CloudProviderID { return ProviderID }

func (p *GCPProvider) Authenticate(ctx context.Context, idp core.IdentityProvider) error {
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

func (p *GCPProvider) IsAuthenticated() bool { return p.ok }
func (p *GCPProvider) AccessToken() string   { return p.token }

func (p *GCPProvider) ListVMs(ctx context.Context) ([]core.VM, error) {
	return nil, core.ErrNotImplemented // Phase 3: Compute aggregatedList
}
func (p *GCPProvider) Connect(ctx context.Context, req core.ConnectRequest) (core.Session, error) {
	return nil, core.ErrNotImplemented // Phase 4
}
func (p *GCPProvider) Logout(ctx context.Context) error { p.token, p.ok = "", false; return nil }

func (p *GCPProvider) getAccount() (core.Account, error) {
	return core.Account{
		ID:    p.cfg.WorkforcePoolUserProject,
		Name:  p.cfg.WorkforcePoolUserProject,
		Roles: []string{p.cfg.Scope},
	}, nil
}
