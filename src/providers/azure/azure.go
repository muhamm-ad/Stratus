// Package azure implements core.ProviderConnector for Azure. Azure is
// natively Entra, so we acquire an ARM-scoped access token silently from the
// Entra identity (via its refresh token), then use that token to list VMs via
// the ARM REST API.
package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/muhamm-ad/stratus/core"
)

const (
	defaultARMBase   = "https://management.azure.com"
	armVMAPIVersion  = "2024-07-01"
)

// Provider implements core.ProviderConnector for Azure.
type Provider struct {
	cfg        Config
	now        func() time.Time
	httpClient *http.Client
	armBase    string

	mu    sync.Mutex
	token string
}

type Option func(*Provider)

func WithClock(f func() time.Time) Option          { return func(p *Provider) { p.now = f } }
func WithHTTPClient(c *http.Client) Option         { return func(p *Provider) { p.httpClient = c } }
func WithARMBase(base string) Option               { return func(p *Provider) { p.armBase = base } }

func New(cfg Config, opts ...Option) *Provider {
	if cfg.ARMScope == "" {
		cfg.ARMScope = DefaultARMScope
	}
	p := &Provider{cfg: cfg, now: time.Now, httpClient: http.DefaultClient, armBase: defaultARMBase}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func (p *Provider) ID() core.ProviderID { return core.ProviderAzure }

func (p *Provider) IsAuthenticated() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.token != ""
}

// Authenticate acquires an ARM-scoped token from the Entra identity, silently.
func (p *Provider) Authenticate(ctx context.Context, idp core.IdentityProvider) error {
	armToken, err := idp.AccessToken(ctx, p.cfg.ARMScope)
	if err != nil {
		return fmt.Errorf("%w: azure acquire ARM token: %v", core.ErrExchange, err)
	}
	p.mu.Lock()
	p.token = armToken
	p.mu.Unlock()
	return nil
}

// AccessToken returns the held ARM token.
func (p *Provider) AccessToken() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.token
}

func (p *Provider) Logout(ctx context.Context) error {
	p.mu.Lock()
	p.token = ""
	p.mu.Unlock()
	return nil
}

// --- Phase 3: ListInstances via ARM REST ---

type armVMListResponse struct {
	Value    []armVM `json:"value"`
	NextLink string  `json:"nextLink"`
}

type armVM struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Location   string     `json:"location"`
	Properties armVMProps `json:"properties"`
}

type armVMProps struct {
	HardwareProfile armHWProfile     `json:"hardwareProfile"`
	StorageProfile  armStorProfile   `json:"storageProfile"`
	InstanceView    *armInstanceView `json:"instanceView,omitempty"`
}

type armHWProfile struct {
	VMSize string `json:"vmSize"`
}

type armStorProfile struct {
	OSDisk armOSDisk `json:"osDisk"`
}

type armOSDisk struct {
	OSType string `json:"osType"` // "Linux" or "Windows"
}

type armInstanceView struct {
	Statuses []armStatus `json:"statuses"`
}

type armStatus struct {
	Code string `json:"code"`
}

// ListInstances lists all VMs under the configured subscription using the ARM API.
// It requests instanceView in a single call to get power state without an extra round-trip.
func (p *Provider) ListInstances(ctx context.Context, _ string) ([]core.Instance, error) {
	p.mu.Lock()
	token := p.token
	p.mu.Unlock()
	if token == "" {
		return nil, core.ErrNotAuthenticated
	}

	startURL := fmt.Sprintf(
		"%s/subscriptions/%s/providers/Microsoft.Compute/virtualMachines?api-version=%s&$expand=instanceView",
		p.armBase, p.cfg.SubscriptionID, armVMAPIVersion,
	)

	var instances []core.Instance
	nextURL := startURL

	for nextURL != "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, nextURL, nil)
		if err != nil {
			return nil, fmt.Errorf("azure: build request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := p.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("azure: list VMs request: %w", err)
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("azure: list VMs returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}

		var page armVMListResponse
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("azure: decode list VMs response: %w", err)
		}
		for _, vm := range page.Value {
			instances = append(instances, mapAzureVM(vm))
		}
		nextURL = page.NextLink
	}
	return instances, nil
}

func mapAzureVM(vm armVM) core.Instance {
	platform := "linux"
	osUser := "azureuser"
	if strings.EqualFold(vm.Properties.StorageProfile.OSDisk.OSType, "Windows") {
		platform = "windows"
		osUser = "Administrator"
	}

	state := "unknown"
	if vm.Properties.InstanceView != nil {
		state = armPowerState(vm.Properties.InstanceView.Statuses)
	}

	return core.Instance{
		ID:           vm.ID,
		Name:         vm.Name,
		State:        state,
		Platform:     platform,
		InstanceType: vm.Properties.HardwareProfile.VMSize,
		PrivateIP:    "", // requires NIC sub-resource call; deferred to Phase 4
		PublicIP:     "",
		Region:       vm.Location,
		OSUser:       osUser,
		Tags:         nil,
	}
}

func armPowerState(statuses []armStatus) string {
	for _, s := range statuses {
		if !strings.HasPrefix(s.Code, "PowerState/") {
			continue
		}
		switch strings.TrimPrefix(s.Code, "PowerState/") {
		case "running":
			return "running"
		case "deallocated", "stopped":
			return "stopped"
		case "deallocating", "starting":
			return "transitioning"
		}
	}
	return "unknown"
}

func (p *Provider) Connect(ctx context.Context, req core.ConnectRequest) (core.Session, error) {
	return nil, core.ErrNotImplemented
}

var _ core.ProviderConnector = (*Provider)(nil)
