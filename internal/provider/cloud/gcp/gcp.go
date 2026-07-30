package gcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google/externalaccount"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"

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
	cfg    Config
	token  string
	ok     bool
	status core.CloudProviderStatus
}

func NewGCPProvider(cfg Config) (*GCPProvider, error) {
	return &GCPProvider{cfg: cfg, status: core.CloudProviderStatusUnauthenticated}, nil
}

func (p *GCPProvider) ID() core.CloudProviderID { return ProviderID }

func (p *GCPProvider) GetStatus() core.CloudProviderStatus {
	if p.status == core.CloudProviderStatusAuthenticated && !p.IsAuthenticated() {
		return core.CloudProviderStatusError
	}
	return p.status
}

func (p *GCPProvider) Authenticate(ctx context.Context, idp core.IdentityProvider) error {
	p.status = core.CloudProviderStatusAuthenticating
	ts, err := externalaccount.NewTokenSource(ctx, externalaccount.Config{
		Audience:                 p.cfg.WorkforceAudience,
		SubjectTokenType:         "urn:ietf:params:oauth:token-type:jwt",
		TokenURL:                 "https://sts.googleapis.com/v1/token",
		Scopes:                   []string{p.cfg.Scope},
		SubjectTokenSupplier:     subjectTokenSupplier{ctx: ctx, idp: idp},
		WorkforcePoolUserProject: p.cfg.WorkforcePoolUserProject,
	})
	if err != nil {
		p.ok = false
		p.status = core.CloudProviderStatusError
		return fmt.Errorf("%w: gcp externalaccount: %v", core.ErrExchange, err)
	}
	tok, err := ts.Token()
	if err != nil {
		p.ok = false
		p.status = core.CloudProviderStatusError
		return fmt.Errorf("%w: gcp STS token-exchange: %v", core.ErrExchange, err)
	}
	p.token, p.ok = tok.AccessToken, true
	p.status = core.CloudProviderStatusAuthenticated
	return nil
}

func (p *GCPProvider) IsAuthenticated() bool { return p.ok }
func (p *GCPProvider) AccessToken() string   { return p.token }

func (p *GCPProvider) ListVMs(ctx context.Context) ([]core.VM, error) {
	if !p.IsAuthenticated() {
		p.status = core.CloudProviderStatusError
		return nil, core.ErrNotAuthenticated
	}
	project := strings.TrimSpace(p.cfg.WorkforcePoolUserProject)
	if project == "" {
		return nil, fmt.Errorf("gcp: workforce_pool_user_project is required to list VMs")
	}

	srv, err := compute.NewService(ctx,
		option.WithTokenSource(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: p.token})),
	)
	if err != nil {
		return nil, fmt.Errorf("gcp: create compute client: %w", err)
	}

	var vms []core.VM
	err = srv.Instances.AggregatedList(project).Pages(ctx, func(page *compute.InstanceAggregatedList) error {
		for zoneURL, scoped := range page.Items {
			zone := zoneName(zoneURL)
			for _, inst := range scoped.Instances {
				if inst == nil {
					continue
				}
				vms = append(vms, mapGCPInstance(inst, zone))
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("gcp: aggregated list instances: %w", err)
	}
	return vms, nil
}

func (p *GCPProvider) Connect(ctx context.Context, req core.ConnectRequest) (core.Session, error) {
	return nil, core.ErrNotImplemented // Phase 4
}
func (p *GCPProvider) Logout(ctx context.Context) error {
	p.token, p.ok = "", false
	p.status = core.CloudProviderStatusUnauthenticated
	return nil
}

func (p *GCPProvider) getAccount() (core.Account, error) {
	return core.Account{
		ID:    p.cfg.WorkforcePoolUserProject,
		Name:  p.cfg.WorkforcePoolUserProject,
		Roles: []string{p.cfg.Scope},
	}, nil
}

func mapGCPInstance(inst *compute.Instance, zone string) core.VM {
	id := inst.Name
	if zone != "" {
		id = zone + "/" + inst.Name
	}
	vm := core.VM{
		ID:       id,
		Name:     inst.Name,
		State:    mapGCPStatus(inst.Status),
		Provider: ProviderID,
		Region:   core.VMRegion(zone),
		Type:     core.VMType(lastPathSegment(inst.MachineType)),
		Tags:     gcpLabels(inst.Labels),
		Platform: core.PlatformLinux,
	}
	if isWindowsInstance(inst) {
		vm.Platform = core.PlatformWindows
		vm.OSUser = "Administrator"
	}
	if inst.CreationTimestamp != "" {
		if t, err := time.Parse(time.RFC3339, inst.CreationTimestamp); err == nil {
			vm.LaunchTime = t
		}
	}
	if len(inst.NetworkInterfaces) > 0 {
		nic := inst.NetworkInterfaces[0]
		if nic.NetworkIP != "" {
			vm.PrivateIP = core.IPAddress(nic.NetworkIP)
		}
		for _, ac := range nic.AccessConfigs {
			if ac != nil && ac.NatIP != "" {
				vm.PublicIP = core.IPAddress(ac.NatIP)
				break
			}
		}
	}
	return vm
}

func mapGCPStatus(status string) core.VMState {
	switch strings.ToUpper(status) {
	case "PROVISIONING", "STAGING":
		return core.StateStarting
	case "RUNNING":
		return core.StateRunning
	case "STOPPING", "SUSPENDING":
		return core.StateStopping
	case "TERMINATED", "SUSPENDED":
		return core.StateStopped
	default:
		return core.StateUnknown
	}
}

func isWindowsInstance(inst *compute.Instance) bool {
	for _, disk := range inst.Disks {
		if disk == nil {
			continue
		}
		for _, lic := range disk.Licenses {
			if strings.Contains(strings.ToLower(lic), "windows") {
				return true
			}
		}
	}
	return false
}

func gcpLabels(labels map[string]string) map[string]string {
	if len(labels) == 0 {
		return nil
	}
	out := make(map[string]string, len(labels))
	for k, v := range labels {
		out[k] = v
	}
	return out
}

func zoneName(zoneURL string) string {
	// "zones/us-central1-a" or full URL ending in /zones/us-central1-a
	if i := strings.LastIndex(zoneURL, "/"); i >= 0 && i+1 < len(zoneURL) {
		return zoneURL[i+1:]
	}
	return zoneURL
}

func lastPathSegment(url string) string {
	if i := strings.LastIndex(url, "/"); i >= 0 && i+1 < len(url) {
		return url[i+1:]
	}
	return url
}
