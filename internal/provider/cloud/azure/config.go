package azure

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Config holds Azure settings. Azure is natively Entra; we only need the
// subscription and the ARM scope to acquire a management token silently.
type Config struct {
	SubscriptionID string `json:"subscription_id"`
	ARMScope       string `json:"arm_scope,omitempty"`
}

const (
	EnvSubscriptionID = "STRATUS_AZURE_SUBSCRIPTION_ID"
	DefaultARMScope   = "https://management.azure.com/.default"
)

func ParseConfig(raw json.RawMessage, getenv func(string) string) (Config, error) {
	var c Config
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &c); err != nil {
			return c, fmt.Errorf("azure: invalid config section: %w", err)
		}
	}
	if v := getenv(EnvSubscriptionID); v != "" {
		c.SubscriptionID = v
	}
	if c.ARMScope == "" {
		c.ARMScope = DefaultARMScope
	}
	return c, c.Validate()
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.SubscriptionID) == "" {
		return fmt.Errorf("azure: incomplete configuration, missing: subscription_id")
	}
	return nil
}
