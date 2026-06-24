// Package core defines the provider-agnostic domain model and the
// CloudConnector contract that every cloud provider (AWS, Azure, GCP)
// implements. It must not import any provider-specific package: the
// dependency direction is always connectors -> core, never the reverse.
package core

import "context"

// ProviderID identifies a cloud provider.
type ProviderID string

const (
	ProviderAWS   ProviderID = "aws"
	ProviderAzure ProviderID = "azure"
	ProviderGCP   ProviderID = "gcp"
)

// CloudConnector is the contract implemented by every provider. The UI and
// orchestration layers depend only on this interface, so adding a new cloud
// means adding a new implementation, not touching callers.
type CloudConnector interface {
	// ID returns the provider this connector talks to.
	ID() ProviderID

	// Login runs the interactive authentication flow (it may open the system
	// browser). It returns when a usable session is established, the user
	// cancels, or ctx is cancelled.
	Login(ctx context.Context) error

	// IsAuthenticated reports whether a non-expired session currently exists.
	IsAuthenticated() bool

	// ListAccounts returns the accounts/entitlements the authenticated user
	// can access. Requires a prior successful Login.
	ListAccounts(ctx context.Context) ([]Account, error)

	// ListInstances returns the connectable instances for one account.
	ListInstances(ctx context.Context, accountID string) ([]Instance, error)

	// Connect opens an interactive session to a single instance.
	Connect(ctx context.Context, req ConnectRequest) (Session, error)

	// Logout clears tokens and any cached credentials.
	Logout(ctx context.Context) error
}
