package tui

import (
	"context"
	"time"

	"github.com/muhamm-ad/stratus/core"
)

// Gateway is the read/act surface the TUI (and, later, other headless facades)
// consume. It depends only on core types. Real implementation is backed by the
// provider plugins THROUGH the service layer; the TUI never sees a provider.
type Gateway interface {
	// Identity (these already exist on the service; listed for completeness).
	IdentityProviders() []IdP
	LoginWith(ctx context.Context, name string, onDeviceCode func(core.DeviceCode)) (Identity, error)
	ActiveIdentity() (Identity, bool)
	ProviderUsable(provider string) (bool, core.IdentityConstraint)

	// VM inventory & sessions (NEW — defined here, mocked below).
	ListVMs(ctx context.Context, provider string) ([]VM, error) // provider "" = all
	OpenSession(ctx context.Context, vmID string) (SessionSpec, error)
	StopVM(ctx context.Context, vmID string) error
	CloseSession(ctx context.Context, sessionID string) error
	Sessions() []Session
	AuditLog() []AuditEntry
	DetectCLIs() []CLIStatus
}

type IdP struct {
	Name        string
	Description string
	Usable      bool
	Constraint  string // e.g. "needs Entra identity"
}

type Identity struct {
	User     string
	IdP      string
	Provider map[string]string // provider -> identity label, e.g. aws:"123456789012 · prod"
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
	When              time.Time
	User, VM, Provider, Method string
	Success           bool
}

type CLIStatus struct {
	Name      string // aws-cli, session-manager-plugin, az-cli, gcloud
	Bin       string // for exec.LookPath
	Detected  bool
	Hint      string // "install to connect"
}
