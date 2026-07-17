// Package core defines the provider-agnostic domain model and the
// CloudProvider contract that every cloud provider (AWS, Azure, GCP)
// implements. It must not import any provider-specific package: the
// dependency direction is always connectors -> core, never the reverse.
package core

import "context"

// CloudProviderID identifies a cloud provider.
type CloudProviderID string

// CloudProvider is the contract implemented by every provider. The UI and
// orchestration layers depend only on this interface, so adding a new cloud
// means adding a new implementation, not touching callers.
type CloudProvider interface {
	// ID returns the provider this cloud provider talks to.
	ID() CloudProviderID

	// Authenticate dérive les identifiants du cloud provider à partir de l'identité
	// déjà établie. Aucun navigateur : appel HTTPS silencieux.
	Authenticate(ctx context.Context, idp IdentityProvider) error

	// IsAuthenticated reports whether a non-expired session currently exists.
	IsAuthenticated() bool

	// ListVMs returns the connectable vms for one account.
	ListVMs(ctx context.Context) ([]VM, error)

	// Connect opens an interactive session to a single vms.
	Connect(ctx context.Context, req ConnectRequest) (Session, error)

	// Logout clears tokens and any cached credentials.
	Logout(ctx context.Context) error
}
