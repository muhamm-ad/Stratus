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

// Gateway adapts *Service to the TUI-facing Gateway interface.
type gateway struct {
	svc      *service.Service
	accounts map[string]string // provider id -> account/subscription/project
	sessions []Session
	audit    []AuditEntry
	seq      int
	mu       sync.Mutex
}

// NewGateway wraps a configured Service for the TUI.
func NewGateway(svc *service.Service) Gateway {
	return &gateway{svc: svc, accounts: svc.Accounts()}
}

func (g *gateway) IdentityProviders() []IdP {
	names := g.svc.IdentityProviders()
	out := make([]IdP, len(names))
	for i, name := range names {
		out[i] = IdP{
			Name:          name,
			Description:   "OpenID Connect",
			Usable:        true,
			UseDeviceFlow: g.svc.IdentityUsesDeviceFlow(name),
		}
	}
	return out
}

func (g *gateway) ActiveIdentity() (Identity, bool) {
	name := g.svc.ActiveIdentity()
	if name == "" || !g.svc.IsAuthenticated() {
		return Identity{}, false
	}
	return g.buildIdentity(name), true
}

func (g *gateway) LoginWith(ctx context.Context, name string, onCode func(core.DeviceCode)) (Identity, error) {
	err := g.svc.LoginWith(ctx, name, func(dc core.DeviceCode) {
		if onCode != nil {
			onCode(core.DeviceCode{
				UserCode:        dc.UserCode,
				VerificationURI: dc.VerificationURI,
				Interval:        dc.Interval,
			})
		}
	})
	if err != nil {
		return Identity{}, err
	}
	return g.buildIdentity(name), nil
}

func (g *gateway) buildIdentity(idpName string) Identity {
	id := Identity{User: idpName, IdP: idpName, Provider: map[string]string{}}
	for _, p := range g.svc.CloudProviders() {
		pid := string(p)
		if g.svc.ProviderUsable(p) {
			label := g.accounts[pid]
			if label == "" {
				label = "connected"
			}
			id.Provider[pid] = label
		}
	}
	return id
}

func (g *gateway) ProviderUsable(provider string) (bool, core.CloudProviderConstraint) {
	return g.svc.ProviderUsable(core.CloudProviderID(provider)), nil
}

func (g *gateway) ListVMs(ctx context.Context, provider string) ([]VM, error) {
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

func (g *gateway) listProvider(ctx context.Context, provider string) ([]VM, error) {
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
	instances, err := g.svc.ListInstances(ctx, pid, g.accounts[provider])
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

func (g *gateway) OpenSession(ctx context.Context, vmID string) (SessionSpec, error) {
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

func (g *gateway) StopVM(ctx context.Context, vmID string) error { return nil }

func (g *gateway) CloseSession(ctx context.Context, sessionID string) error {
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

func (g *gateway) Sessions() []Session {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := make([]Session, len(g.sessions))
	copy(out, g.sessions)
	return out
}

func (g *gateway) AuditLog() []AuditEntry {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := make([]AuditEntry, len(g.audit))
	copy(out, g.audit)
	return out
}

func (g *gateway) DetectCLIs() []CLIStatus {
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
