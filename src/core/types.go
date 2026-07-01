package core

import "time"

// Account is a cloud account/subscription/project the user can access.
type Account struct {
	ID    string   // AWS account ID, Azure subscription ID, GCP project ID
	Name  string   // human-readable name
	Roles []string // roles/permission sets available in this account
}

// Instance is a connectable virtual machine, normalized across providers.
type Instance struct {
	ID           string
	Name         string
	State        string // running, stopped, pending, ...
	Platform     string // "linux" or "windows"
	InstanceType string // e.g. t3.large, Standard_D4s_v5, n2-standard-4
	PrivateIP    string
	PublicIP     string
	Region       string
	OSUser       string // default OS login user, when known
	LaunchTime   time.Time
	Tags         map[string]string
}

// IsRunning reports whether the instance can currently accept connections.
func (i Instance) IsRunning() bool { return i.State == "running" }

// IsWindows reports whether the instance runs Windows (RDP) vs Linux (SSH).
func (i Instance) IsWindows() bool { return i.Platform == "windows" }

// ConnectMethod selects how Stratus opens a session to an instance.
type ConnectMethod string

const (
	// ConnectSSMShell opens an interactive shell via Session Manager.
	ConnectSSMShell ConnectMethod = "ssm-shell"
	// ConnectSSMSSH tunnels SSH over a Session Manager proxy.
	ConnectSSMSSH ConnectMethod = "ssm-ssh"
	// ConnectSSMRDP port-forwards RDP over Session Manager.
	ConnectSSMRDP ConnectMethod = "ssm-rdp"
)

// ConnectRequest describes a single connection attempt.
type ConnectRequest struct {
	AccountID string
	Role      string
	Instance  Instance
	Method    ConnectMethod
}

// Session represents a live connection to an instance, backed by a launched
// process (e.g. the AWS CLI session-manager-plugin or an SSH client).
type Session interface {
	// Wait blocks until the underlying process exits.
	Wait() error
	// Close terminates the session early.
	Close() error
}
