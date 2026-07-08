package oidc

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Config describes any OIDC provider. Minimal form: {issuer, client_id}.
type Config struct {
	Issuer            string   `json:"issuer,omitempty"`
	ClientID          string   `json:"client_id"`
	Scopes            []string `json:"scopes,omitempty"`
	AuthorizeEndpoint string   `json:"authorize_endpoint,omitempty"`
	TokenEndpoint     string   `json:"token_endpoint,omitempty"`
	// UseDeviceFlow forces RFC 8628 (headless / no browser) instead of loopback.
	UseDeviceFlow bool `json:"use_device_flow,omitempty"`
}

func (c Config) scopes() []string {
	if len(c.Scopes) == 0 {
		return []string{"openid", "offline_access"}
	}
	return c.Scopes
}

func (c Config) hasEndpoints() bool { return c.AuthorizeEndpoint != "" && c.TokenEndpoint != "" }

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
		return fmt.Errorf("oidc: set either \"issuer\" (discovery) or both endpoints")
	}
	return nil
}