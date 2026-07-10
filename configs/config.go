// Package configs owns the single, build-time-embedded configuration file and
// exposes it as raw, named sections. It is deliberately generic: it knows
// nothing about which identity providers or cloud providers exist. Each
// component parses its own section (a json.RawMessage), so a new provider is
// added by writing its own package — configs never changes.
//
// The file lives here (config/config.json) because //go:embed can only
// reference files in the embedding package's own directory or a subdirectory,
// never a parent. The committed default ships empty so a clean checkout builds.
package configs

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed myconfig.json
var raw []byte

// Sections holds the raw configuration grouped into identity providers and
// cloud providers, each keyed by name. Values stay as json.RawMessage so every
// component owns its own schema and this package stays decoupled from them.
type Sections struct {
	Identity  map[string]json.RawMessage `json:"identity"`
	Providers map[string]json.RawMessage `json:"providers"`
}

// Load parses the embedded configuration into its named sections.
func Load() (Sections, error) {
	var s Sections
	if err := json.Unmarshal(raw, &s); err != nil {
		return s, fmt.Errorf("config: invalid config.json: %w", err)
	}
	return s, nil
}

// IdentitySection returns the raw section for a named identity provider
// (e.g. "entra"), or nil if absent.
func (s Sections) IdentitySection(name string) json.RawMessage { return s.Identity[name] }

// ProviderSection returns the raw section for a named cloud provider
// (e.g. "aws"), or nil if absent.
func (s Sections) ProviderSection(id string) json.RawMessage { return s.Providers[id] }
