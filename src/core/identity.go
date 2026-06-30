package core

import "context"

// IdentityProvider performs the single interactive sign-in (Microsoft Entra ID)
// and hands out the tokens that connectors exchange for cloud-specific
// credentials. One browser login, then silent
// derivation per provider.
type IdentityProvider interface {
	// Login opens the browser ONCE (PKCE + loopback) and caches the Entra
	// id_token / access_token / refresh_token.
	Login(ctx context.Context) error

	// IsAuthenticated reports whether a valid Entra session exists.
	IsAuthenticated() bool

	// IDToken returns the Entra id_token (JWT, aud = client_id), refreshing it
	// silently if needed. Used by AWS (AssumeRoleWithWebIdentity) and GCP
	// (STS token exchange).
	IDToken(ctx context.Context) (string, error)

	// AccessToken silently acquires an access token for an arbitrary scope
	// (e.g. the ARM scope for Azure) via the refresh token. No browser.
	AccessToken(ctx context.Context, scope string) (string, error)

	// Logout clears the cached Entra session.
	Logout(ctx context.Context) error
}