package gcp

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Config holds GCP Workforce Identity Federation settings.
type Config struct {
	// WorkforceAudience is the STS audience, e.g.
	// //iam.googleapis.com/locations/global/workforcePools/POOL/providers/PROV
	WorkforceAudience string `json:"workforce_audience"`
	// ProjectID is the GCP project ID used to list Compute instances.
	ProjectID string `json:"project_id"`
	// Scope defaults to cloud-platform when empty.
	Scope string `json:"scope,omitempty"`
}

const (
	EnvAudience   = "STRATUS_GCP_WORKFORCE_AUDIENCE"
	EnvProjectID  = "STRATUS_GCP_PROJECT_ID"
	DefaultScope  = "https://www.googleapis.com/auth/cloud-platform"
)

func ParseConfig(raw json.RawMessage, getenv func(string) string) (Config, error) {
	var c Config
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &c); err != nil {
			return c, fmt.Errorf("gcp: invalid config section: %w", err)
		}
	}
	if v := getenv(EnvAudience); v != "" {
		c.WorkforceAudience = v
	}
	if v := getenv(EnvProjectID); v != "" {
		c.ProjectID = v
	}
	if c.Scope == "" {
		c.Scope = DefaultScope
	}
	return c, c.Validate()
}

func (c Config) Validate() error {
	var missing []string
	if strings.TrimSpace(c.WorkforceAudience) == "" {
		missing = append(missing, "workforce_audience")
	}
	if strings.TrimSpace(c.ProjectID) == "" {
		missing = append(missing, "project_id")
	}
	if len(missing) > 0 {
		return fmt.Errorf("gcp: incomplete configuration, missing: %s", strings.Join(missing, ", "))
	}
	return nil
}
