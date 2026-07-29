package core

import (
	"context"
	"time"
)

// IdentityProviderID is a unique identifier for an identity provider.
type IdentityProviderID string

// DeviceCode is surfaced to the UI during the RFC 8628 device-authorization
// fallback (headless / loopback-blocked environments).
type DeviceCode struct {
	UserCode        string
	VerificationURI string
	Interval        time.Duration
}

// IdentityProvider is any OIDC identity (Entra, Okta, Keycloak, …).
// All token acquisition/refresh is delegated to golang.org/x/oauth2.
type IdentityProvider interface {

	// ID returns the ID of the identity provider.
	ID() IdentityProviderID

	// Login opens the browser ONCE (PKCE + loopback) and caches the Entra
	// id_token / access_token / refresh_token.
	// onCode is called only when the device flow is used (nil-safe); the browser flow ignores it.
	Login(ctx context.Context, onCode func(DeviceCode)) error
	
	// IsAuthenticated reports whether a valid OIDC session exists.
	IsAuthenticated() bool
	
	// IDToken returns a valid (refreshed if needed) id_token for federation.
	IDToken(ctx context.Context) (string, error)

	// AccessToken silently acquires an access token for the given scopes via the
	// refresh token (e.g. the Azure ARM scope). Only meaningful when the issuer
	// can grant those scopes.
	AccessToken(ctx context.Context, scopes ...string) (string, error)
	
	// Logout invalidates the OIDC session.
	Logout(ctx context.Context) error

	// UserInfo returns the user info for the current session.
	UserInfo(ctx context.Context) (map[string]string, error)
}