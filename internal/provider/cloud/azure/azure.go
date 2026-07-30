package azure

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v8"

	"github.com/muhamm-ad/stratus/internal/core"
)

const ProviderID core.CloudProviderID = "azure"

// useMockListVMs routes ListVMs to MockListVMs for UI testing.
var useMockListVMs = true

// AzureProvider acquires an ARM-scoped token from the ACTIVE identity's refresh
// token (delegated to x/oauth2 inside the auth.Session). This preserves the
// single sign-on: no second browser, no separate Azure login. ARM only accepts
// Entra-issued tokens, hence AcceptsIssuer (core.IdentityConstraint).
type AzureProvider struct {
	cfg      Config
	armToken string
	ok       bool
	status   core.CloudProviderStatus

	mu sync.Mutex
}

func NewAzureProvider(cfg Config) (*AzureProvider, error) {
	if cfg.ARMScope == "" {
		cfg.ARMScope = DefaultARMScope
	}
	return &AzureProvider{cfg: cfg, status: core.CloudProviderStatusUnauthenticated}, nil
}

func (p *AzureProvider) ID() core.CloudProviderID { return ProviderID }

func (p *AzureProvider) GetStatus() core.CloudProviderStatus {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.status == core.CloudProviderStatusAuthenticated && !p.ok {
		return core.CloudProviderStatusError
	}
	return p.status
}

func (p *AzureProvider) Authenticate(ctx context.Context, idp core.IdentityProvider) error {
	p.mu.Lock()
	p.status = core.CloudProviderStatusAuthenticating
	p.mu.Unlock()

	tok, err := idp.AccessToken(ctx, p.cfg.ARMScope)
	if err != nil {
		p.mu.Lock()
		p.ok = false
		p.status = core.CloudProviderStatusError
		p.mu.Unlock()
		return fmt.Errorf("%w: azure ARM token: %v", core.ErrExchange, err)
	}
	p.mu.Lock()
	p.armToken, p.ok = tok, true
	p.status = core.CloudProviderStatusAuthenticated
	p.mu.Unlock()
	return nil
}

func (p *AzureProvider) IsAuthenticated() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.ok
}

func (p *AzureProvider) AccessToken() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.armToken
}

// AcceptsIssuer implements core.IdentityConstraint: ARM requires an Entra token.
func (p *AzureProvider) AcceptsIssuer(issuer string) bool {
	return strings.Contains(issuer, "login.microsoftonline.com") ||
		strings.Contains(issuer, "sts.windows.net")
}

// MockListVMs returns a fixed set for UI testing.
func (p *AzureProvider) MockListVMs(ctx context.Context) ([]core.VM, error) {
	const sub = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Compute/virtualMachines/"
	now := time.Now()
	return []core.VM{
		{
			ID: sub + "dev-stratus-l-1", Name: "dev-stratus-l-1", State: core.StateRunning,
			Provider: ProviderID, Tags: map[string]string{"environment": "dev"},
			Region: core.VMRegion("eastus"), Type: core.VMType("Standard_B1s"),
			Platform: core.PlatformLinux, OSUser: "azureuser",
			PrivateIP: "10.0.0.4", PublicIP: "20.1.2.3", LaunchTime: now.Add(-24 * time.Hour),
		},
		{
			ID: sub + "dev-stratus-w-1", Name: "dev-stratus-w-1", State: core.StateRunning,
			Provider: ProviderID, Tags: map[string]string{"environment": "dev"},
			Region: core.VMRegion("eastus"), Type: core.VMType("Standard_B2s"),
			Platform: core.PlatformWindows, OSUser: "azureuser",
			PrivateIP: "10.0.0.5", PublicIP: "20.1.2.4", LaunchTime: now.Add(-48 * time.Hour),
		},
		{
			ID: sub + "staging-api-1", Name: "staging-api-1", State: core.StateStopped,
			Provider: ProviderID, Tags: map[string]string{"environment": "staging"},
			Region: core.VMRegion("westus2"), Type: core.VMType("Standard_D2s_v3"),
			Platform: core.PlatformLinux, OSUser: "azureuser",
			PrivateIP: "10.1.0.10", LaunchTime: now.Add(-72 * time.Hour),
		},
		{
			ID: sub + "prod-web-1", Name: "prod-web-1", State: core.StateRunning,
			Provider: ProviderID, Tags: map[string]string{"environment": "prod"},
			Region: core.VMRegion("eastus"), Type: core.VMType("Standard_B1ms"),
			Platform: core.PlatformLinux, OSUser: "azureuser",
			PrivateIP: "10.2.0.20", PublicIP: "40.10.20.30", LaunchTime: now.Add(-120 * time.Hour),
		},
	}, nil
}

func (p *AzureProvider) ListVMs(ctx context.Context) ([]core.VM, error) {
	// Temporary: use mock data for UI testing. Flip to false to hit Azure.
	if useMockListVMs {
		return p.MockListVMs(ctx)
	}

	if !p.IsAuthenticated() {
		p.mu.Lock()
		p.status = core.CloudProviderStatusError
		p.mu.Unlock()
		return nil, core.ErrNotAuthenticated
	}

	client, err := armcompute.NewVirtualMachinesClient(p.cfg.SubscriptionID, bearerTokenCredential{token: p.AccessToken()}, nil)
	if err != nil {
		return nil, fmt.Errorf("azure: create compute client: %w", err)
	}

	// Full list for metadata, then statusOnly for power state (ListAll has no $expand).
	vmsByID := make(map[string]core.VM)
	pager := client.NewListAllPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("azure: list VMs: %w", err)
		}
		for _, vm := range page.Value {
			if vm == nil {
				continue
			}
			if deleting(vm) {
				continue
			}
			mapped := mapAzureVM(vm)
			vmsByID[mapped.ID] = mapped
		}
	}

	statusPager := client.NewListAllPager(&armcompute.VirtualMachinesClientListAllOptions{
		StatusOnly: to.Ptr("true"),
	})
	for statusPager.More() {
		page, err := statusPager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("azure: list VM status: %w", err)
		}
		for _, vm := range page.Value {
			if vm == nil || vm.ID == nil {
				continue
			}
			id := strings.ToLower(*vm.ID)
			existing, ok := vmsByID[id]
			if !ok {
				continue
			}
			existing.State = mapAzurePowerState(vm)
			vmsByID[id] = existing
		}
	}

	out := make([]core.VM, 0, len(vmsByID))
	for _, vm := range vmsByID {
		out = append(out, vm)
	}
	return out, nil
}

func (p *AzureProvider) Connect(ctx context.Context, req core.ConnectRequest) (core.Session, error) {
	return nil, core.ErrNotImplemented // Phase 4
}

func (p *AzureProvider) Logout(ctx context.Context) error {
	p.mu.Lock()
	p.armToken, p.ok = "", false
	p.status = core.CloudProviderStatusUnauthenticated
	p.mu.Unlock()
	return nil
}

func (p *AzureProvider) getAccount() (core.Account, error) {
	return core.Account{
		ID:    p.cfg.SubscriptionID,
		Name:  p.cfg.SubscriptionID,
		Roles: []string{p.cfg.ARMScope},
	}, nil
}

// bearerTokenCredential adapts the ARM access token already acquired via SSO
// into the azcore.TokenCredential interface expected by the ARM SDK.
type bearerTokenCredential struct {
	token string
}

func (c bearerTokenCredential) GetToken(_ context.Context, _ policy.TokenRequestOptions) (azcore.AccessToken, error) {
	return azcore.AccessToken{
		Token:     c.token,
		ExpiresOn: time.Now().Add(time.Hour),
	}, nil
}

func deleting(vm *armcompute.VirtualMachine) bool {
	if vm.Properties == nil || vm.Properties.ProvisioningState == nil {
		return false
	}
	return strings.EqualFold(*vm.Properties.ProvisioningState, "Deleting")
}

func mapAzureVM(vm *armcompute.VirtualMachine) core.VM {
	id := ""
	if vm.ID != nil {
		id = strings.ToLower(*vm.ID)
	}
	name := id
	if vm.Name != nil && *vm.Name != "" {
		name = *vm.Name
	}

	out := core.VM{
		ID:       id,
		Name:     name,
		State:    mapAzurePowerState(vm),
		Provider: ProviderID,
		Tags:     azureTags(vm.Tags),
	}
	if vm.Location != nil {
		out.Region = core.VMRegion(*vm.Location)
	}
	if vm.Properties != nil {
		if vm.Properties.HardwareProfile != nil && vm.Properties.HardwareProfile.VMSize != nil {
			out.Type = core.VMType(string(*vm.Properties.HardwareProfile.VMSize))
		}
		if vm.Properties.StorageProfile != nil && vm.Properties.StorageProfile.OSDisk != nil &&
			vm.Properties.StorageProfile.OSDisk.OSType != nil {
			switch *vm.Properties.StorageProfile.OSDisk.OSType {
			case armcompute.OperatingSystemTypesWindows:
				out.Platform = core.PlatformWindows
				out.OSUser = "Administrator"
			default:
				out.Platform = core.PlatformLinux
				out.OSUser = "azureuser"
			}
		}
		if vm.Properties.TimeCreated != nil {
			out.LaunchTime = *vm.Properties.TimeCreated
		}
	}
	if out.Platform == "" {
		out.Platform = core.PlatformLinux
		out.OSUser = "azureuser"
	}
	return out
}

func mapAzurePowerState(vm *armcompute.VirtualMachine) core.VMState {
	if vm.Properties == nil || vm.Properties.InstanceView == nil {
		return core.StateUnknown
	}
	for _, st := range vm.Properties.InstanceView.Statuses {
		if st == nil || st.Code == nil {
			continue
		}
		code := strings.ToLower(*st.Code)
		if !strings.HasPrefix(code, "powerstate/") {
			continue
		}
		switch strings.TrimPrefix(code, "powerstate/") {
		case "running":
			return core.StateRunning
		case "starting":
			return core.StateStarting
		case "stopping", "deallocating":
			return core.StateStopping
		case "stopped", "deallocated":
			return core.StateStopped
		default:
			return core.StateUnknown
		}
	}
	return core.StateUnknown
}

func azureTags(tags map[string]*string) map[string]string {
	if len(tags) == 0 {
		return nil
	}
	out := make(map[string]string, len(tags))
	for k, v := range tags {
		if v != nil {
			out[k] = *v
		}
	}
	return out
}
