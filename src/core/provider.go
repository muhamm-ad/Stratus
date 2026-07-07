// Package core defines the provider-agnostic domain model and the
// ProviderConnector contract that every cloud provider (AWS, Azure, GCP)
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

// ProviderConnector is the contract implemented by every provider. The UI and
// orchestration layers depend only on this interface, so adding a new cloud
// means adding a new implementation, not touching callers.
type ProviderConnector interface {
	// ID returns the provider this connector talks to.
	ID() ProviderID

	// Authenticate dérive les identifiants du provider à partir de l'identité
	// déjà établie. Aucun navigateur : appel HTTPS silencieux.
	Authenticate(ctx context.Context, idp IdentityProvider) error

	// IsAuthenticated reports whether a non-expired session currently exists.
	IsAuthenticated() bool

	// ListInstances returns the connectable vms for one account.
	ListInstances(ctx context.Context, accountID string) ([]Instance, error)

	// Connect opens an interactive session to a single vms.
	Connect(ctx context.Context, req ConnectRequest) (Session, error)

	// Logout clears tokens and any cached credentials.
	Logout(ctx context.Context) error

	// GetAccount returns the account/subscription/project ID from the connector's configuration.
	GetAccount() (string, error)
}
