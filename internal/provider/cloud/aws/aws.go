package aws

import (
	"context"
	"fmt"
	"time"

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

// useMockListVMs routes ListVMs to MockListVMs for UI testing.
var useMockListVMs = true

// AWSProvider federates the OIDC id_token to AWS via STS
// AssumeRoleWithWebIdentity, using the SDK's built-in web-identity provider
// (handles caching + refresh).
type AWSProvider struct {
	cfg    Config
	awsCfg awssdk.Config
	creds  awssdk.Credentials
	ok     bool
	status core.CloudProviderStatus
}

func NewAWSProvider(cfg Config) (*AWSProvider, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(awssdk.AnonymousCredentials{}),
	)
	if err != nil {
		return nil, fmt.Errorf("aws: config: %w", err)
	}
	return &AWSProvider{
		cfg:    cfg,
		awsCfg: awsCfg,
		status: core.CloudProviderStatusUnauthenticated,
	}, nil
}

func (p *AWSProvider) ID() core.CloudProviderID { return ProviderID }

func (p *AWSProvider) GetStatus() core.CloudProviderStatus {
	if p.status == core.CloudProviderStatusAuthenticated && !p.IsAuthenticated() {
		return core.CloudProviderStatusError
	}
	return p.status
}

func (p *AWSProvider) Authenticate(ctx context.Context, idp core.IdentityProvider) error {
	p.status = core.CloudProviderStatusAuthenticating
	provider := stscreds.NewWebIdentityRoleProvider(
		sts.NewFromConfig(p.awsCfg),
		p.cfg.RoleArn,
		idTokenRetriever{ctx: ctx, idp: idp},
	)
	creds, err := provider.Retrieve(ctx)
	if err != nil {
		p.ok = false
		p.status = core.CloudProviderStatusError
		return fmt.Errorf("%w: aws AssumeRoleWithWebIdentity: %v", core.ErrExchange, err)
	}
	p.creds = creds
	p.awsCfg.Credentials = credentials.NewStaticCredentialsProvider(
		creds.AccessKeyID,
		creds.SecretAccessKey,
		creds.SessionToken,
	)
	p.ok = true
	p.status = core.CloudProviderStatusAuthenticated
	return nil
}

func (p *AWSProvider) IsAuthenticated() bool           { return p.ok && !p.creds.Expired() }
func (p *AWSProvider) Credentials() awssdk.Credentials { return p.creds }

// MockListVMs returns a fixed set for UI testing.
func (p *AWSProvider) MockListVMs(ctx context.Context) ([]core.VM, error) {
	region := core.VMRegion(p.cfg.Region)
	now := time.Now()
	return []core.VM{
		{
			ID: "i-0abc123def4567890", Name: "dev-stratus-linux-1", State: core.StateRunning,
			Provider: ProviderID, Tags: map[string]string{"Name": "dev-stratus-linux-1", "environment": "dev"},
			Region: region, Type: core.VMType("t3.micro"), Platform: core.PlatformLinux, OSUser: "ec2-user",
			PrivateIP: "10.0.1.10", PublicIP: "3.92.195.172", LaunchTime: now.Add(-24 * time.Hour),
		},
		{
			ID: "i-0bcd234ef5678901a", Name: "dev-stratus-windows-1", State: core.StateRunning,
			Provider: ProviderID, Tags: map[string]string{"Name": "dev-stratus-windows-1", "environment": "dev"},
			Region: region, Type: core.VMType("t3.small"), Platform: core.PlatformWindows, OSUser: "Administrator",
			PrivateIP: "10.0.1.11", PublicIP: "13.221.124.118", LaunchTime: now.Add(-48 * time.Hour),
		},
		{
			ID: "i-0cde345fg6789012b", Name: "staging-api-1", State: core.StateStopped,
			Provider: ProviderID, Tags: map[string]string{"Name": "staging-api-1", "environment": "staging"},
			Region: region, Type: core.VMType("t3.medium"), Platform: core.PlatformLinux, OSUser: "ec2-user",
			PrivateIP: "10.0.2.20", LaunchTime: now.Add(-72 * time.Hour),
		},
		{
			ID: "i-0def456gh7890123c", Name: "staging-worker-1", State: core.StateRunning,
			Provider: ProviderID, Tags: map[string]string{"Name": "staging-worker-1", "environment": "staging"},
			Region: region, Type: core.VMType("t3.large"), Platform: core.PlatformLinux, OSUser: "ubuntu",
			PrivateIP: "10.0.2.21", PublicIP: "54.10.20.30", LaunchTime: now.Add(-96 * time.Hour),
		},
		{
			ID: "i-0efg567hi8901234d", Name: "prod-db-1", State: core.StateRunning,
			Provider: ProviderID, Tags: map[string]string{"Name": "prod-db-1", "environment": "prod"},
			Region: region, Type: core.VMType("m5.large"), Platform: core.PlatformLinux, OSUser: "ec2-user",
			PrivateIP: "10.0.3.30", LaunchTime: now.Add(-168 * time.Hour),
		},
		{
			ID: "i-0fgh678ij9012345e", Name: "prod-bastion-1", State: core.StateStopped,
			Provider: ProviderID, Tags: map[string]string{"Name": "prod-bastion-1", "environment": "prod"},
			Region: region, Type: core.VMType("t3.nano"), Platform: core.PlatformLinux, OSUser: "ec2-user",
			PrivateIP: "10.0.3.31", PublicIP: "18.200.1.50", LaunchTime: now.Add(-200 * time.Hour),
		},
	}, nil
}

func (p *AWSProvider) ListVMs(ctx context.Context) ([]core.VM, error) {
	// Temporary: use mock data for UI testing. Flip to false to hit AWS.
	if useMockListVMs {
		return p.MockListVMs(ctx)
	}

	if !p.IsAuthenticated() {
		p.status = core.CloudProviderStatusError
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
	p.status = core.CloudProviderStatusUnauthenticated
	return nil
}

// func (p *AWSProvider) getAccount() (core.Account, error) {
// 	parts := strings.Split(p.cfg.RoleArn, ":")
// 	if len(parts) >= 5 {
// 		return core.Account{
// 			ID:    parts[4],
// 			Name:  parts[4],
// 			Roles: []string{p.cfg.RoleArn},
// 		}, nil
// 	}
// 	return core.Account{}, fmt.Errorf("aws: error getting account from role ARN")
// }

