package app

import (
	"context"
	"sort"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/muhamm-ad/stratus/internal/core"
	"github.com/muhamm-ad/stratus/internal/service"
)

// App is the Wails-bound desktop adapter. It holds NO orchestration logic — it
// delegates to *service.Service.
type App struct {
	ctx context.Context
	svc *service.Service
}

func NewApp(svc *service.Service) *App { return &App{svc: svc} }

// Startup is called by Wails with the application context.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

// onDeviceCode forwards an RFC 8628 device code to the frontend as a Wails
// event. It fires only when a provider uses the device flow; the default
// browser + loopback flow never calls it. The React side listens with
// runtime.EventsOn("auth:device-code", ...).
func (a *App) onDeviceCode(dc core.DeviceCode) {
	wruntime.EventsEmit(a.ctx, "auth:device-code", map[string]any{
		"userCode":        dc.UserCode,
		"verificationURI": dc.VerificationURI,
		"intervalSeconds": dc.Interval.Seconds(),
	})
}

// ── Identity choice for single sign-on (bound to the frontend) ──────────────

// IdentityProviders lists the configured OIDC provider IDs (e.g. "entra","okta").
func (a *App) IdentityProviders() []string {
	ids := a.svc.IdentityProvidersIDs()
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = string(id)
	}
	return out
}

// LoginWith signs in using the chosen identity provider (browser once) and
// silently authenticates every cloud provider. Blocking: Wails runs bound
// methods off the UI thread, so the frontend just awaits it.
func (a *App) LoginWith(identityProviderID string) error {
	_, err, _ := a.svc.LoginWith(a.ctx, core.IdentityProviderID(identityProviderID), a.onDeviceCode)
	return err
}

// ActiveIdentityProviderID returns the signed-in identity provider ID, or "".
func (a *App) ActiveIdentityProviderID() string {
	return string(a.svc.GetActiveIdentityProviderID())
}

// Login is a convenience for the single-provider case.
func (a *App) Login() error {
	_, err, _ := a.svc.Login(a.ctx, a.onDeviceCode)
	return err
}

func (a *App) IsAuthenticated() bool { return a.svc.IsAuthenticated() }
func (a *App) Logout() error         { return a.svc.Logout(a.ctx) }

// CloudProviders lists registered cloud-provider IDs (e.g. "aws","azure","gcp").
func (a *App) CloudProviders() []string {
	ids := a.svc.GetCloudProvidersIDs()
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = string(id)
	}
	return out
}

// ProviderUsable reports whether a cloud provider works with the active identity
// (e.g. Azure needs an Entra identity). Lets the UI gray out incompatible ones.
func (a *App) ProviderUsable(cloudProviderID string) bool {
	return a.svc.ProviderUsable(core.CloudProviderID(cloudProviderID))
}

// Connect derives a cloud provider's credentials from the active identity.
// func (a *App) Connect(cloudProviderID string) error {
// 	return a.svc.AuthenticateCloudProvider(a.ctx, core.CloudProviderID(cloudProviderID))
// }

// VMData is the JSON-serialisable VM shape sent to the React frontend.
// Field tags match the TypeScript VMInstance interface in types/domain.ts.
type VMData struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Provider     string   `json:"provider"`
	Region       string   `json:"region"`
	State        string   `json:"state"`
	Platform     string   `json:"platform"`
	InstanceType string   `json:"type"` // "size" matches the UI column label
	PrivateIP    string   `json:"privateIP"`
	PublicIP     string   `json:"publicIP"`
	OSUser       string   `json:"osUser"`
	Tags         []string `json:"tags"`       // ["key:value", ...]
	CanConnect   bool     `json:"canConnect"` // derived: state == "running"
}

func toVMData(inst core.VM, provider core.CloudProviderID) VMData {
	tags := make([]string, 0, len(inst.Tags))
	for k, v := range inst.Tags {
		tags = append(tags, k+":"+v)
	}
	sort.Strings(tags)
	return VMData{
		ID:           inst.ID,
		Name:         inst.Name,
		Provider:     string(provider),
		Region:       string(inst.Region),
		State:        string(inst.State),
		Platform:     string(inst.Platform),
		InstanceType: string(inst.Type),
		PrivateIP:    string(inst.PrivateIP),
		PublicIP:     string(inst.PublicIP),
		OSUser:       string(inst.OSUser),
		Tags:         tags,
		CanConnect:   inst.IsRunning(),
	}
}

// ListVMs returns connectable VMs for one cloud provider.
func (a *App) ListVMs(cloudProviderID core.CloudProviderID) ([]VMData, error) {
	instances, err := a.svc.ListVMs(a.ctx, cloudProviderID)
	if err != nil {
		return nil, err
	}
	out := make([]VMData, len(instances))
	for i, inst := range instances {
		out[i] = toVMData(inst, cloudProviderID)
	}
	return out, nil
}
