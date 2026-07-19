package aws

import (
	"context"
	"fmt"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
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
	cfg    Config
	awsCfg awssdk.Config
	creds  awssdk.Credentials
	ok     bool
}

func NewAWSProvider(cfg Config) (*AWSProvider, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(awssdk.AnonymousCredentials{}),
	)
	if err != nil {
		return nil, fmt.Errorf("aws: config: %w", err)
	}
	return &AWSProvider{cfg: cfg, awsCfg: awsCfg}, nil
}

func (p *AWSProvider) ID() core.CloudProviderID { return ProviderID }

func (p *AWSProvider) Authenticate(ctx context.Context, idp core.IdentityProvider) error {
	provider := stscreds.NewWebIdentityRoleProvider(
		sts.NewFromConfig(p.awsCfg),
		p.cfg.RoleArn,
		idTokenRetriever{ctx: ctx, idp: idp},
	)
	creds, err := provider.Retrieve(ctx)
	if err != nil {
		return fmt.Errorf("%w: aws AssumeRoleWithWebIdentity: %v", core.ErrExchange, err)
	}
	p.creds = creds
	p.awsCfg.Credentials = credentials.NewStaticCredentialsProvider(
		creds.AccessKeyID,
		creds.SecretAccessKey,
		creds.SessionToken,
	)
	p.ok = true
	return nil
}

func (p *AWSProvider) IsAuthenticated() bool           { return p.ok && !p.creds.Expired() }
func (p *AWSProvider) Credentials() awssdk.Credentials { return p.creds }

func (p *AWSProvider) ListVMs(ctx context.Context) ([]core.VM, error) {
	if !p.IsAuthenticated() {
		return nil, core.ErrNotAuthenticated
	}

	client := ec2.NewFromConfig(p.awsCfg)
	var vms []core.VM
	paginator := ec2.NewDescribeInstancesPaginator(client, &ec2.DescribeInstancesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("aws: describe instances: %w", err)
		}
		for _, res := range page.Reservations {
			for _, inst := range res.Instances {
				if inst.State != nil {
					switch inst.State.Name {
					case ec2types.InstanceStateNameShuttingDown, ec2types.InstanceStateNameTerminated:
						continue
					}
				}
				vms = append(vms, mapEC2Instance(inst, p.cfg.Region))
			}
		}
	}
	return vms, nil
}

func mapEC2Instance(inst ec2types.Instance, region string) core.VM {
	vm := core.VM{
		ID:       awssdk.ToString(inst.InstanceId),
		State:    mapEC2State(inst.State),
		Provider: ProviderID,
		Region:   core.VMRegion(region),
		Type:     core.VMType(inst.InstanceType),
		Tags:     ec2Tags(inst.Tags),
	}
	if inst.LaunchTime != nil {
		vm.LaunchTime = *inst.LaunchTime
	}
	if inst.PrivateIpAddress != nil {
		vm.PrivateIP = core.IPAddress(*inst.PrivateIpAddress)
	}
	if inst.PublicIpAddress != nil {
		vm.PublicIP = core.IPAddress(*inst.PublicIpAddress)
	}
	if inst.Platform == ec2types.PlatformValuesWindows {
		vm.Platform = core.PlatformWindows
		vm.OSUser = "Administrator"
	} else {
		vm.Platform = core.PlatformLinux
		vm.OSUser = "ec2-user"
	}
	if name := vm.Tags["Name"]; name != "" {
		vm.Name = name
	} else {
		vm.Name = vm.ID
	}
	return vm
}

func mapEC2State(state *ec2types.InstanceState) core.VMState {
	if state == nil {
		return core.StateUnknown
	}
	switch state.Name {
	case ec2types.InstanceStateNamePending:
		return core.StateStarting
	case ec2types.InstanceStateNameRunning:
		return core.StateRunning
	case ec2types.InstanceStateNameStopping:
		return core.StateStopping
	case ec2types.InstanceStateNameStopped:
		return core.StateStopped
	default:
		return core.StateUnknown
	}
}

func ec2Tags(tags []ec2types.Tag) map[string]string {
	if len(tags) == 0 {
		return nil
	}
	out := make(map[string]string, len(tags))
	for _, t := range tags {
		if t.Key != nil && t.Value != nil {
			out[*t.Key] = *t.Value
		}
	}
	return out
}

func (p *AWSProvider) Connect(ctx context.Context, req core.ConnectRequest) (core.Session, error) {
	return nil, core.ErrNotImplemented // Phase 4
}

func (p *AWSProvider) Logout(ctx context.Context) error {
	p.creds, p.ok = awssdk.Credentials{}, false
	p.awsCfg.Credentials = awssdk.AnonymousCredentials{}
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
