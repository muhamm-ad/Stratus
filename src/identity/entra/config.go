package entra

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Config holds the (non-secret) Entra public-client settings. A desktop public
// client (RFC 8252) holds NO secret — PKCE plus the user's IdP auth provide the
// security — and the redirect URI is a runtime loopback, so neither is here.
type Config struct {
	TenantID string `json:"tenant_id"`
	ClientID string `json:"client_id"`
}

const (
	EnvTenantID = "STRATUS_ENTRA_TENANT_ID"
	EnvClientID = "STRATUS_ENTRA_CLIENT_ID"
)

// ParseConfig builds a Config from this identity provider's section, with env
// overrides (dev/CI). The entra package owns this schema; config never sees it.
func ParseConfig(raw json.RawMessage, getenv func(string) string) (Config, error) {
	var c Config
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &c); err != nil {
			return c, fmt.Errorf("entra: invalid config section: %w", err)
		}
	}
	if v := getenv(EnvTenantID); v != "" {
		c.TenantID = v
	}
	if v := getenv(EnvClientID); v != "" {
		c.ClientID = v
	}
	return c, c.Validate()
}

func (c Config) Validate() error {
	var missing []string
	if c.TenantID == "" {
		missing = append(missing, "tenant_id")
	}
	if c.ClientID == "" {
		missing = append(missing, "client_id")
	}
	if len(missing) > 0 {
		return fmt.Errorf("entra: incomplete configuration, missing: %s", strings.Join(missing, ", "))
	}
	return nil
}