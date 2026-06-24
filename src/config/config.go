// Package config owns the single, build-time-embedded configuration file that
// carries the settings for every cloud connector. It is deliberately a leaf
// package: it knows nothing about the connector packages. Each connector parses
// its own section (a json.RawMessage), which keeps the schema for a provider's
// settings inside that provider's package and avoids coupling config to AWS,
// Azure, or GCP.
//
// The file lives here (config/config.json) rather than at the repository root
// because //go:embed can only reference files in the embedding package's own
// directory or a subdirectory — never a parent. The enterprise packager
// overwrites config/config.json before `wails build`; the committed default
// ships empty so a clean checkout always builds.
package config

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed config.json
var raw []byte

// Sections holds the raw, per-provider configuration. Storing each provider's
// settings as json.RawMessage lets every connector own its own schema while
// this package stays decoupled from them.
type Sections struct {
	AWS   json.RawMessage `json:"aws"`
	Azure json.RawMessage `json:"azure"`
	GCP   json.RawMessage `json:"gcp"`
}

// Load parses the embedded configuration into its provider sections. A missing
// top-level key yields a nil RawMessage for that provider, which connectors
// treat as "no embedded values" (they can still be configured via environment).
func Load() (Sections, error) {
	var s Sections
	if err := json.Unmarshal(raw, &s); err != nil {
		return s, fmt.Errorf("config: invalid config.json: %w", err)
	}
	return s, nil
}
