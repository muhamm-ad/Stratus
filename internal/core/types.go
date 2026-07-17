package core

// Account is a cloud account/subscription/project the user can access.
type Account struct {
	ID    string   // AWS account ID, Azure subscription ID, GCP project ID
	Name  string   // human-readable name
	Roles []string // roles/permission sets available in this account
}

type Role string

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
	Account Account
	Role    Role
	VM      VM
	Method  ConnectMethod
}

// Session represents a live connection to an instance, backed by a launched
// process (e.g. the AWS CLI session-manager-plugin or an SSH client).
type Session interface {
	// Wait blocks until the underlying process exits.
	Wait() error
	// Close terminates the session early.
	Close() error
}
