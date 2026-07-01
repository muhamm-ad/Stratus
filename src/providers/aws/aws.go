// Package aws implements core.ProviderConnector for AWS: it exchanges
// the Entra id_token for temporary AWS credentials via STS
// AssumeRoleWithWebIdentity, then uses those credentials to list EC2 instances.
package aws

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	ststypes "github.com/aws/aws-sdk-go-v2/service/sts/types"

	"github.com/muhamm-ad/stratus/core"
)

// assumer is the subset of the STS client used here (mockable in tests).
type assumer interface {
	AssumeRoleWithWebIdentity(context.Context, *sts.AssumeRoleWithWebIdentityInput, ...func(*sts.Options)) (*sts.AssumeRoleWithWebIdentityOutput, error)
}

// ec2lister is the subset of the EC2 client used for listing instances (mockable in tests).
type ec2lister interface {
	DescribeInstances(context.Context, *ec2.DescribeInstancesInput, ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
}

// Provider implements core.ProviderConnector for AWS.
type Provider struct {
	cfg    Config
	newSTS func(region string) assumer
	newEC2 func(region string, creds awssdk.CredentialsProvider) ec2lister
	now    func() time.Time

	mu    sync.Mutex
	creds *ststypes.Credentials
}

type Option func(*Provider)

// WithAssumerFactory injects a custom STS client factory (tests).
func WithAssumerFactory(f func(region string) assumer) Option {
	return func(p *Provider) { p.newSTS = f }
}

// WithEC2ListerFactory injects a custom EC2 client factory (tests).
func WithEC2ListerFactory(f func(region string, creds awssdk.CredentialsProvider) ec2lister) Option {
	return func(p *Provider) { p.newEC2 = f }
}

func WithClock(f func() time.Time) Option { return func(p *Provider) { p.now = f } }

// New builds an AWS connector.
func New(cfg Config, opts ...Option) *Provider {
	p := &Provider{
		cfg:    cfg,
		newSTS: defaultAssumer,
		newEC2: defaultEC2Lister,
		now:    time.Now,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// defaultAssumer builds an STS client. AssumeRoleWithWebIdentity is unsigned
// (the web identity token is the credential), so anonymous credentials are used.
func defaultAssumer(region string) assumer {
	return sts.New(sts.Options{Region: region, Credentials: awssdk.AnonymousCredentials{}})
}

func defaultEC2Lister(region string, creds awssdk.CredentialsProvider) ec2lister {
	return ec2.New(ec2.Options{Region: region, Credentials: creds})
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
		RoleArn:          awssdk.String(p.cfg.RoleArn),
		RoleSessionName:  awssdk.String("stratus"),
		WebIdentityToken: awssdk.String(idToken),
	})
	if err != nil {
		return fmt.Errorf("%w: aws assume role with web identity: %v", core.ErrExchange, err)
	}
	p.mu.Lock()
	p.creds = out.Credentials
	p.mu.Unlock()
	return nil
}

// Credentials returns the currently held temporary AWS credentials.
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

// ListInstances returns all EC2 instances in the configured region using the
// temporary credentials obtained during Authenticate.
func (p *Provider) ListInstances(ctx context.Context, _ string) ([]core.Instance, error) {
	p.mu.Lock()
	creds := p.creds
	p.mu.Unlock()
	if creds == nil {
		return nil, core.ErrNotAuthenticated
	}

	staticCreds := awssdk.CredentialsProviderFunc(func(_ context.Context) (awssdk.Credentials, error) {
		p.mu.Lock()
		c := p.creds
		p.mu.Unlock()
		if c == nil {
			return awssdk.Credentials{}, core.ErrNotAuthenticated
		}
		return awssdk.Credentials{
			AccessKeyID:     awssdk.ToString(c.AccessKeyId),
			SecretAccessKey: awssdk.ToString(c.SecretAccessKey),
			SessionToken:    awssdk.ToString(c.SessionToken),
			Expires:         awssdk.ToTime(c.Expiration),
		}, nil
	})

	client := p.newEC2(p.cfg.Region, staticCreds)

	var instances []core.Instance
	var nextToken *string
	for {
		out, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("aws: describe instances: %w", err)
		}
		for _, r := range out.Reservations {
			for _, i := range r.Instances {
				instances = append(instances, mapEC2Instance(i, p.cfg.Region))
			}
		}
		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}
	return instances, nil
}

func mapEC2Instance(i ec2types.Instance, region string) core.Instance {
	tags := make(map[string]string, len(i.Tags))
	var name string
	for _, t := range i.Tags {
		k, v := awssdk.ToString(t.Key), awssdk.ToString(t.Value)
		tags[k] = v
		if k == "Name" {
			name = v
		}
	}

	platform := "linux"
	if i.Platform == ec2types.PlatformValuesWindows {
		platform = "windows"
	}
	// Detect Amazon Linux / Ubuntu default users from tags or AMI name.
	// Fall back to "ec2-user" (Amazon Linux default).
	osUser := "ec2-user"
	if platform == "windows" {
		osUser = "Administrator"
	} else if u, ok := tags["stratus:os-user"]; ok {
		osUser = u
	} else if strings.EqualFold(tags["os"], "ubuntu") {
		osUser = "ubuntu"
	}

	state := "unknown"
	if i.State != nil {
		state = string(i.State.Name)
	}

	return core.Instance{
		ID:           awssdk.ToString(i.InstanceId),
		Name:         name,
		State:        state,
		Platform:     platform,
		InstanceType: string(i.InstanceType),
		PrivateIP:    awssdk.ToString(i.PrivateIpAddress),
		PublicIP:     awssdk.ToString(i.PublicIpAddress),
		Region:       region,
		OSUser:       osUser,
		LaunchTime:   awssdk.ToTime(i.LaunchTime),
		Tags:         tags,
	}
}

func (p *Provider) Connect(ctx context.Context, req core.ConnectRequest) (core.Session, error) {
	return nil, core.ErrNotImplemented
}

var _ core.ProviderConnector = (*Provider)(nil)
