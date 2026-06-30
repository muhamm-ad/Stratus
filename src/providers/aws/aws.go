// Package aws implements core.ProviderConnector for AWS: it exchanges
// the Entra id_token for temporary AWS credentials via STS
// AssumeRoleWithWebIdentity.
package aws

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	ststypes "github.com/aws/aws-sdk-go-v2/service/sts/types"

	"github.com/muhamm-ad/stratus/core"
)

// assumer is the subset of the STS client used here (mockable in tests).
type assumer interface {
	AssumeRoleWithWebIdentity(context.Context, *sts.AssumeRoleWithWebIdentityInput, ...func(*sts.Options)) (*sts.AssumeRoleWithWebIdentityOutput, error)
}

// Provider implements core.ProviderConnector for AWS.
type Provider struct {
	cfg    Config
	newSTS func(region string) assumer
	now    func() time.Time

	mu    sync.Mutex
	creds *ststypes.Credentials
}

type Option func(*Provider)

// WithAssumerFactory injects a custom STS client factory (tests).
func WithAssumerFactory(f func(region string) assumer) Option {
	return func(p *Provider) { p.newSTS = f }
}
func WithClock(f func() time.Time) Option { return func(p *Provider) { p.now = f } }

// New builds an AWS connector.
func New(cfg Config, opts ...Option) *Provider {
	p := &Provider{cfg: cfg, newSTS: defaultAssumer, now: time.Now}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// defaultAssumer builds an STS client. AssumeRoleWithWebIdentity is unsigned
// (the web identity token is the credential), so anonymous credentials are used.
func defaultAssumer(region string) assumer {
	return sts.New(sts.Options{Region: region, Credentials: aws.AnonymousCredentials{}})
}

func (p *Provider) ID() core.ProviderID { return core.ProviderAWS }

func (p *Provider) IsAuthenticated() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.creds == nil || p.creds.Expiration == nil {
		return false
	}
	return p.now().Before(*p.creds.Expiration)
}

// Authenticate exchanges the Entra id_token for temporary AWS credentials.
func (p *Provider) Authenticate(ctx context.Context, idp core.IdentityProvider) error {
	idToken, err := idp.IDToken(ctx)
	if err != nil {
		return fmt.Errorf("aws: %w", err)
	}
	out, err := p.newSTS(p.cfg.Region).AssumeRoleWithWebIdentity(ctx, &sts.AssumeRoleWithWebIdentityInput{
		RoleArn:          aws.String(p.cfg.RoleArn),
		RoleSessionName:  aws.String("stratus"),
		WebIdentityToken: aws.String(idToken),
	})
	if err != nil {
		return fmt.Errorf("%w: aws assume role with web identity: %v", core.ErrExchange, err)
	}
	p.mu.Lock()
	p.creds = out.Credentials
	p.mu.Unlock()
	return nil
}

// Credentials returns the currently held temporary AWS credentials (used later
// by the connection layer to populate the environment for the session).
func (p *Provider) Credentials() *ststypes.Credentials {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.creds
}

func (p *Provider) Logout(ctx context.Context) error {
	p.mu.Lock()
	p.creds = nil
	p.mu.Unlock()
	return nil
}

// --- Phase 3 / 4 ---
func (p *Provider) ListInstances(ctx context.Context, accountID string) ([]core.Instance, error) {
	return nil, core.ErrNotImplemented
}
func (p *Provider) Connect(ctx context.Context, req core.ConnectRequest) (core.Session, error) {
	return nil, core.ErrNotImplemented
}

var _ core.ProviderConnector = (*Provider)(nil)
