// Package auth wraps the standard OIDC/OAuth2 libraries for a desktop public
// client (RFC 8252 browser+loopback+PKCE, with RFC 8628 device fallback). It
// replaces the previously hand-rolled PKCE, loopback server, discovery, and
// token-endpoint calls with:
//
//	github.com/coreos/go-oidc/v3  — discovery + id_token verification (JWKS)
//	golang.org/x/oauth2           — PKCE, token grants, auto-refresh, device flow
//	github.com/int128/oauth2cli   — the browser + loopback dance (RFC 8252)
//	github.com/zalando/go-keyring — token persistence
package auth

import (
	"context"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// Client bundles a discovered OIDC provider, an id_token verifier, and an
// oauth2.Config for a public client (no secret).
type Client struct {
	Provider *oidc.Provider
	Verifier *oidc.IDTokenVerifier
	OAuth    oauth2.Config
}

// NewClient discovers endpoints from {issuer}/.well-known/openid-configuration.
func NewClient(ctx context.Context, issuer, clientID string, scopes []string) (*Client, error) {
	p, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery %q: %w", issuer, err)
	}
	return &Client{
		Provider: p,
		Verifier: p.Verifier(&oidc.Config{ClientID: clientID}),
		OAuth: oauth2.Config{
			ClientID: clientID,
			Endpoint: p.Endpoint(),
			Scopes:   scopes,
		},
	}, nil
}

// NewClientManual builds a Client from explicit endpoints (providers without a
// discovery document). id_token signature verification is skipped unless a
// verifier is wired separately; issuer is only used for error context.
func NewClientManual(clientID, authorizeEP, tokenEP string, scopes []string) *Client {
	return &Client{
		OAuth: oauth2.Config{
			ClientID: clientID,
			Endpoint: oauth2.Endpoint{AuthURL: authorizeEP, TokenURL: tokenEP},
			Scopes:   scopes,
		},
	}
}