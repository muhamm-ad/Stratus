package aws

import (
	"context"
	"fmt"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/muhamm-ad/stratus/internal/core"
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

const ProviderID core.CloudProviderID = "aws"

// AWSProvider federates the OIDC id_token to AWS via STS
// AssumeRoleWithWebIdentity, using the SDK's built-in web-identity provider
// (handles caching + refresh).
type AWSProvider struct {
	cfg   Config
	creds awssdk.Credentials
	ok    bool
}

func New(cfg Config) *AWSProvider { return &AWSProvider{cfg: cfg} }

func (p *AWSProvider) ID() core.CloudProviderID { return ProviderID }

func (p *AWSProvider) Authenticate(ctx context.Context, idp core.IdentityProvider) error {
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

func (p *AWSProvider) IsAuthenticated() bool           { return p.ok && !p.creds.Expired() }
func (p *AWSProvider) Credentials() awssdk.Credentials { return p.creds }

func (p *AWSProvider) ListVMs(ctx context.Context) ([]core.VM, error) {
	return nil, core.ErrNotImplemented // Phase 3: EC2 DescribeInstances
}
func (p *AWSProvider) Connect(ctx context.Context, req core.ConnectRequest) (core.Session, error) {
	return nil, core.ErrNotImplemented // Phase 4
}
func (p *AWSProvider) Logout(ctx context.Context) error {
	p.creds, p.ok = awssdk.Credentials{}, false
	return nil
}

func (p *AWSProvider) getAccount() (core.Account, error) {
	parts := strings.Split(p.cfg.RoleArn, ":")
	if len(parts) >= 5 {
		return core.Account{
			ID:    parts[4],
			Name:  parts[4],
			Roles: []string{p.cfg.RoleArn},
		}, nil
	}
	return core.Account{}, fmt.Errorf("aws: error getting account from role ARN")
}
