package main

import (
	"context"
	"fmt"
	"sort"

	"github.com/muhamm-ad/stratus/bootstrap"
	"github.com/muhamm-ad/stratus/core"
	"github.com/muhamm-ad/stratus/service"
)

// App is the Wails-bound desktop adapter. It holds NO orchestration logic — it
// delegates everything to *service.Service, the same shared logic the CLI
// (cmd/stratus-cli) uses. Both UIs differ only in presentation.
type App struct {
	ctx context.Context
	svc *service.Service
}

func NewApp() *App { return &App{} }

// startup is called by Wails with the application context.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	svc, warnings, err := bootstrap.New() // same assembly the CLI calls
	if err != nil {
		panic(fmt.Sprintf("bootstrap: %v", err))
	}
	for _, w := range warnings {
		fmt.Printf("provider unavailable: %v\n", w) // incomplete config → skipped
	}
	a.svc = svc
}

// ── Methods bound to the React frontend (thin delegation) ───────────────────

func (a *App) Login() error          { return a.svc.Login(a.ctx) }
func (a *App) IsAuthenticated() bool { return a.svc.IsAuthenticated() }
func (a *App) Logout() error         { return a.svc.Logout(a.ctx) }

func (a *App) Providers() []string {
	ids := a.svc.Providers()
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = string(id)
	}
	return out
}

// ConnectProvider derives a provider's credentials from the Entra session —
// silent, no browser. Call after Login().
func (a *App) ConnectProvider(id string) error {
	return a.svc.Connect(a.ctx, core.ProviderID(id))
}

// VMData is the JSON-serialisable VM shape sent to the React frontend.
// Named fields match the TypeScript VMInstance interface in types/domain.ts.
type VMData struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Provider     string   `json:"provider"`
	Region       string   `json:"region"`
	State        string   `json:"state"`
	Platform     string   `json:"platform"`
	InstanceType string   `json:"size"`       // "size" matches the UI column label
	PrivateIP    string   `json:"privateIP"`
	PublicIP     string   `json:"publicIP"`
	OSUser       string   `json:"osUser"`
	Tags         []string `json:"tags"`       // ["key:value", ...]
	CanConnect   bool     `json:"canConnect"` // derived: state == "running"
}

func toVMData(inst core.Instance, provider string) VMData {
	tags := make([]string, 0, len(inst.Tags))
	for k, v := range inst.Tags {
		tags = append(tags, k+":"+v)
	}
	sort.Strings(tags)
	return VMData{
		ID:           inst.ID,
		Name:         inst.Name,
		Provider:     provider,
		Region:       inst.Region,
		State:        inst.State,
		Platform:     inst.Platform,
		InstanceType: inst.InstanceType,
		PrivateIP:    inst.PrivateIP,
		PublicIP:     inst.PublicIP,
		OSUser:       inst.OSUser,
		Tags:         tags,
		CanConnect:   inst.IsRunning(),
	}
}

// ListInstances returns connectable VMs for one provider. Phase 3.
// accountID may be empty — providers that support multiple accounts use it to
// scope the query; single-account providers ignore it.
func (a *App) ListInstances(providerID, accountID string) ([]VMData, error) {
	instances, err := a.svc.ListInstances(a.ctx, core.ProviderID(providerID), accountID)
	if err != nil {
		return nil, err
	}
	out := make([]VMData, len(instances))
	for i, inst := range instances {
		out[i] = toVMData(inst, providerID)
	}
	return out, nil
}
