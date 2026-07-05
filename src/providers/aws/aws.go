package aws

import (
	"context"
	"fmt"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/muhamm-ad/stratus/core"
)

// idTokenRetriever adapts the active identity to the AWS SDK's
// stscreds.IdentityTokenRetriever interface.
type idTokenRetriever struct {
	ctx context.Context
	idp core.IdentityProvider
}

func (r idTokenRetriever) GetIdentityToken() ([]byte, error) {
	t, err := r.idp.IDToken(r.ctx)
	if err != nil {
		return nil, err
	}
	return []byte(t), nil
}

// Provider federates the OIDC id_token to AWS via STS
// AssumeRoleWithWebIdentity, using the SDK's built-in web-identity provider
// (handles caching + refresh).
type Provider struct {
	cfg   Config
	creds awssdk.Credentials
	ok    bool
}

func New(cfg Config) *Provider { return &Provider{cfg: cfg} }

func (p *Provider) ID() core.ProviderID { return core.ProviderAWS }

func (p *Provider) Authenticate(ctx context.Context, idp core.IdentityProvider) error {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(p.cfg.Region),
		awsconfig.WithCredentialsProvider(awssdk.AnonymousCredentials{}), // web-identity call is unsigned
	)
	if err != nil {
		return fmt.Errorf("%w: aws config: %v", core.ErrExchange, err)
	}
	provider := stscreds.NewWebIdentityRoleProvider(
		sts.NewFromConfig(awsCfg),
		p.cfg.RoleArn,
		idTokenRetriever{ctx: ctx, idp: idp},
	)
	creds, err := provider.Retrieve(ctx)
	if err != nil {
		return fmt.Errorf("%w: aws AssumeRoleWithWebIdentity: %v", core.ErrExchange, err)
	}
	p.creds, p.ok = creds, true
	return nil
}

func (p *Provider) IsAuthenticated() bool           { return p.ok && !p.creds.Expired() }
func (p *Provider) Credentials() awssdk.Credentials { return p.creds }

func (p *Provider) ListInstances(ctx context.Context, account string) ([]core.Instance, error) {
	return nil, core.ErrNotImplemented // Phase 3: EC2 DescribeInstances
}
func (p *Provider) Connect(ctx context.Context, req core.ConnectRequest) (core.Session, error) {
	return nil, core.ErrNotImplemented // Phase 4
}
func (p *Provider) Logout(ctx context.Context) error {
	p.creds, p.ok = awssdk.Credentials{}, false
	return nil
}
