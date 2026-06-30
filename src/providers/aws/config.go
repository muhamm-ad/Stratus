package aws

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Config holds the AWS web-identity settings. None are secret.
type Config struct {
	RoleArn string `json:"role_arn"`
	Region  string `json:"region"`
}

const (
	EnvRoleArn = "STRATUS_AWS_ROLE_ARN"
	EnvRegion  = "STRATUS_AWS_REGION"
)

// ParseConfig builds an AWS Config from this provider's config section, with
// env overrides. The aws package owns this schema; config never sees it.
func ParseConfig(raw json.RawMessage, getenv func(string) string) (Config, error) {
	var c Config
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &c); err != nil {
			return c, fmt.Errorf("aws: invalid config section: %w", err)
		}
	}
	if v := getenv(EnvRoleArn); v != "" {
		c.RoleArn = v
	}
	if v := getenv(EnvRegion); v != "" {
		c.Region = v
	}
	return c, c.Validate()
}

func (c Config) Validate() error {
	var missing []string
	if c.RoleArn == "" {
		missing = append(missing, "role_arn")
	}
	if c.Region == "" {
		missing = append(missing, "region")
	}
	if len(missing) > 0 {
		return fmt.Errorf("aws: incomplete configuration, missing: %s", strings.Join(missing, ", "))
	}
	return nil
}
