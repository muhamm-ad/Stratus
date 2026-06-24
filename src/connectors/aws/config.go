package aws

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Config holds the (non-secret) AWS SSO settings. None of these are sensitive:
// a public OAuth client cannot hold a usable secret, so security comes from
// PKCE plus the user's own IdP authentication, not from any value here.
type Config struct {
	SSOStartURL   string `json:"sso_start_url"`
	SSORegion     string `json:"sso_region"`
	DefaultRegion string `json:"default_region"`
	PreferredRole string `json:"preferred_role,omitempty"`
}

// Environment variable names that override embedded values (dev / CI / tests).
const (
	EnvStartURL      = "STRATUS_AWS_SSO_START_URL"
	EnvSSORegion     = "STRATUS_AWS_SSO_REGION"
	EnvDefaultRegion = "STRATUS_AWS_DEFAULT_REGION"
	EnvPreferredRole = "STRATUS_AWS_PREFERRED_ROLE"
)

// ParseConfig builds an AWS Config from this provider's section of the central
// configuration (see package config). The section is the single source of
// truth; environment variables then override individual fields for dev/CI.
// Resolution is last-writer-wins:
//
//	embedded section  ->  environment variables
//
// The connector owns its own schema, parsing, and validation here, so the
// config package never needs to import this one.
func ParseConfig(raw json.RawMessage) (Config, error) {
	var c Config
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &c); err != nil {
			return c, fmt.Errorf("aws: invalid config section: %w", err)
		}
	}

	if v := os.Getenv(EnvStartURL); v != "" {
		c.SSOStartURL = v
	}
	if v := os.Getenv(EnvSSORegion); v != "" {
		c.SSORegion = v
	}
	if v := os.Getenv(EnvDefaultRegion); v != "" {
		c.DefaultRegion = v
	}
	if v := os.Getenv(EnvPreferredRole); v != "" {
		c.PreferredRole = v
	}

	return c, c.validate()
}

func (c Config) validate() error {
	var missing []string
	if c.SSOStartURL == "" {
		missing = append(missing, "sso_start_url")
	}
	if c.SSORegion == "" {
		missing = append(missing, "sso_region")
	}
	if c.DefaultRegion == "" {
		missing = append(missing, "default_region")
	}
	if len(missing) > 0 {
		return fmt.Errorf("aws: incomplete configuration, missing: %s", strings.Join(missing, ", "))
	}
	return nil
}