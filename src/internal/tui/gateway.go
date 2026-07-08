package tui

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/muhamm-ad/stratus/core"
	"github.com/muhamm-ad/stratus/service"
)

type GatwayeIdentityProvider struct {
	Name          string
	Description   string
	Usable        bool
	UseDeviceFlow bool   // true → RFC 8628; false → browser+loopback (default)
	Constraint    string // e.g. "needs Entra identity"
}

type VMState string

const (
	StateRunning  VMState = "running"
	StateStopped  VMState = "stopped"
	StateStarting VMState = "starting"
	StateStopping VMState = "stopping"
	StateUnknown  VMState = "unknown"
)

type VM struct {
	Name, ID, Provider, Region, Type string
	State                            VMState
	PrivateIP                        string
	Method                           string // SSM / Bastion / IAP
	Tags                             map[string]string
	CanConnect                       bool
	Recent                           []Activity
}

type Activity struct {
	OK   bool
	When time.Time
	Text string
}

// SessionSpec is what the TUI turns into an *exec.Cmd. The service decides the
// method+args; the TUI just runs it via tea.ExecProcess. This keeps the CLI
// invocation logic in the (provider-aware) service layer, not in the UI.
type SessionSpec struct {
	SessionID string
	VMName    string
	Provider  string
	Bin       string   // "aws" | "az" | "gcloud"
	Args      []string // full argv
}

type Session struct {
	ID, Target, Provider, Method string
	Opened                       time.Time
}

type AuditEntry struct {
	When                       time.Time
	User, VM, Provider, Method string
	Success                    bool
}

type CLIStatus struct {
	Name     string // aws-cli, session-manager-plugin, az-cli, gcloud
	Bin      string // for exec.LookPath
	Detected bool
	Hint     string // "install to connect"
}

type Gateway struct {
	svc      *service.Service
	sessions []Session
	audit    []AuditEntry
	seq      int
	mu       sync.Mutex
}

// NewGateway wraps a configured Service for the TUI.
func NewGateway(svc *service.Service) *Gateway {
	return &Gateway{svc: svc}
}

func (g *Gateway) IdentityProviders() []GatwayeIdentityProvider {
	ids := g.svc.IdentityProvidersIDs()
	out := make([]GatwayeIdentityProvider, len(ids))
	for i, id := range ids {
		out[i] = GatwayeIdentityProvider{
			Name:          string(id),
			Description:   "OpenID Connect",
			Usable:        true,
			UseDeviceFlow: g.svc.IdentityUsesDeviceFlow(id),
		}
	}
	return out
}

// REVIEW: Check if this is needed
// ---- VM inventory & sessions ----------------------------------------------

func (g *Gateway) ListVMs(ctx context.Context, provider string) ([]VM, error) {
	if provider == "" {
		var all []VM
		for _, id := range g.svc.CloudProviders() {
			vms, err := g.listProvider(ctx, string(id))
			if err != nil {
				return nil, err
			}
			all = append(all, vms...)
		}
		return all, nil
	}
	return g.listProvider(ctx, provider)
}

func (g *Gateway) listProvider(ctx context.Context, provider string) ([]VM, error) {
	pid := core.CloudProviderID(provider)
	if !g.svc.ProviderUsable(pid) {
		return nil, fmt.Errorf("%w: %s incompatible with active identity", core.ErrExchange, provider)
	}
	if err := g.svc.Connect(ctx, pid); err != nil {
		if errors.Is(err, core.ErrNotAuthenticated) || errors.Is(err, core.ErrExchange) {
			return nil, err
		}
		// Config/connection issues: return empty, not fatal.
		return nil, nil
	}
	instances, err := g.svc.ListInstances(ctx, pid, g.svc.Accounts()[pid])
	if err != nil {
		if errors.Is(err, core.ErrNotImplemented) {
			return nil, nil
		}
		if errors.Is(err, core.ErrNotAuthenticated) || errors.Is(err, core.ErrExchange) {
			return nil, err
		}
		return nil, nil
	}
	out := make([]VM, len(instances))
	for i, inst := range instances {
		out[i] = instanceToVM(inst, provider)
	}
	return out, nil
}

func instanceToVM(inst core.Instance, provider string) VM {
	method := "SSM"
	switch provider {
	case "azure":
		method = "Bastion"
	case "gcp":
		method = "IAP"
	}
	return VM{
		Name:       inst.Name,
		ID:         inst.ID,
		Provider:   provider,
		Region:     inst.Region,
		Type:       inst.InstanceType,
		State:      mapInstanceState(inst.State),
		PrivateIP:  inst.PrivateIP,
		Method:     method,
		Tags:       inst.Tags,
		CanConnect: inst.IsRunning(),
	}
}

func mapInstanceState(s string) VMState {
	switch strings.ToLower(s) {
	case "running":
		return StateRunning
	case "stopped", "terminated":
		return StateStopped
	case "pending", "starting":
		return StateStarting
	case "stopping":
		return StateStopping
	default:
		return StateUnknown
	}
}

// BuildSessionSpec constructs the native CLI argv for connecting to a VM.
// Provider-specific knowledge lives here so the TUI only runs tea.ExecProcess.
func BuildSessionSpec(vm VM, sessionID string) SessionSpec {
	spec := SessionSpec{
		SessionID: sessionID,
		VMName:    vm.Name,
		Provider:  vm.Provider,
	}
	switch vm.Provider {
	case "aws":
		spec.Bin, spec.Args = "aws", []string{
			"ssm", "start-session", "--target", vm.ID, "--region", vm.Region,
		}
	case "azure":
		spec.Bin, spec.Args = "az", []string{
			"network", "bastion", "ssh",
			"--name", "stratus-bastion", "--resource-group", "stratus-prod",
			"--target-resource-id", vm.ID, "--auth-type", "AAD",
		}
	case "gcp":
		spec.Bin, spec.Args = "gcloud", []string{
			"compute", "ssh", vm.Name,
			"--tunnel-through-iap", "--zone=" + vm.Region + "-a", "--project=stratus-dev",
		}
	}
	if spec.Bin != "" {
		if _, err := exec.LookPath(spec.Bin); err != nil {
			spec.Bin, spec.Args = "sh", []string{"-c",
				fmt.Sprintf("echo 'stratus: connected to %s via %s. type exit to return.'; exec ${SHELL:-sh}",
					vm.Name, vm.Method)}
		}
	}
	return spec
}

func (g *Gateway) OpenSession(ctx context.Context, vmID string) (SessionSpec, error) {
	vms, err := g.ListVMs(ctx, "")
	if err != nil {
		return SessionSpec{}, err
	}
	var vm VM
	for _, v := range vms {
		if v.ID == vmID || v.Name == vmID {
			vm = v
			break
		}
	}
	if vm.ID == "" {
		return SessionSpec{}, fmt.Errorf("gateway: vm %q not found", vmID)
	}
	g.mu.Lock()
	g.seq++
	id := fmt.Sprintf("sess-%d", g.seq)
	g.mu.Unlock()
	spec := BuildSessionSpec(vm, id)
	g.mu.Lock()
	g.sessions = append(g.sessions, Session{
		ID: id, Target: vm.Name, Provider: vm.Provider, Method: vm.Method, Opened: time.Now(),
	})
	g.mu.Unlock()
	return spec, nil
}

func (g *Gateway) StopVM(ctx context.Context, vmID string) error { return nil }

func (g *Gateway) CloseSession(ctx context.Context, sessionID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	for i, s := range g.sessions {
		if s.ID == sessionID {
			g.sessions = append(g.sessions[:i], g.sessions[i+1:]...)
			break
		}
	}
	return nil
}

func (g *Gateway) Sessions() []Session {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := make([]Session, len(g.sessions))
	copy(out, g.sessions)
	return out
}

func (g *Gateway) AuditLog() []AuditEntry {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := make([]AuditEntry, len(g.audit))
	copy(out, g.audit)
	return out
}

func (g *Gateway) DetectCLIs() []CLIStatus {
	det := func(name, bin, hint string) CLIStatus {
		_, err := exec.LookPath(bin)
		return CLIStatus{Name: name, Bin: bin, Detected: err == nil, Hint: hint}
	}
	return []CLIStatus{
		det("aws-cli", "aws", "install to connect"),
		det("session-manager-plugin", "session-manager-plugin", "install to connect"),
		det("az-cli", "az", "install to connect"),
		det("gcloud", "gcloud", "install to connect"),
	}
}
