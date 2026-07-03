// Package oidc is a single, generic OpenID Connect identity provider. Because
// Entra, Okta, Keycloak, Auth0, … all speak the same protocol, Stratus does not
// need one package per vendor: each identity provider is just a named entry
// under "identity" in config.json, and this one implementation serves them all.
package oidc

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Config describes any OIDC provider. Minimal form is {issuer, client_id}; the
// authorize/token endpoints are then discovered from
// {issuer}/.well-known/openid-configuration.
type Config struct {
	// Issuer is the OIDC issuer URL. Examples:
	//   Entra:    https://login.microsoftonline.com/<tenant-id>/v2.0
	//   Okta:     https://<domain>/oauth2/<authServer>   (authServer often "default")
	//   Keycloak: https://<host>/realms/<realm>
	//   Auth0:    https://<tenant>.auth0.com/
	Issuer string `json:"issuer,omitempty"`

	// ClientID is the public client registered with the provider.
	ClientID string `json:"client_id"`

	// Scopes requested at login. Defaults to ["openid","offline_access"]
	// (offline_access yields the refresh token used for silent federation).
	Scopes []string `json:"scopes,omitempty"`

	// AuthorizeEndpoint/TokenEndpoint override discovery when both are set —
	// for providers without a discovery document, or to pin endpoints.
	AuthorizeEndpoint string `json:"authorize_endpoint,omitempty"`
	TokenEndpoint     string `json:"token_endpoint,omitempty"`
}

func (c Config) scopes() []string {
	if len(c.Scopes) == 0 {
		return []string{"openid", "offline_access"}
	}
	return c.Scopes
}

func (c Config) scopeString() string { return strings.Join(c.scopes(), " ") }

func (c Config) hasEndpoints() bool { return c.AuthorizeEndpoint != "" && c.TokenEndpoint != "" }

// ParseConfig decodes and validates one identity section.
func ParseConfig(raw json.RawMessage) (Config, error) {
	var c Config
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &c); err != nil {
			return c, fmt.Errorf("oidc: invalid config section: %w", err)
		}
	}
	return c, c.Validate()
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.ClientID) == "" {
		return fmt.Errorf("oidc: missing client_id")
	}
	if strings.TrimSpace(c.Issuer) == "" && !c.hasEndpoints() {
		return fmt.Errorf("oidc: set either \"issuer\" (for discovery) or both \"authorize_endpoint\" and \"token_endpoint\"")
	}
	return nil
}