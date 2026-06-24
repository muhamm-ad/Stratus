package config

import (
	"encoding/json"
	"testing"
)

func TestLoad_ReturnsProviderSections(t *testing.T) {
	secs, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	// The committed config.json ships an (empty) aws section; it must at least
	// be present and be valid JSON.
	if len(secs.AWS) == 0 {
		t.Fatal("expected a non-empty raw aws section")
	}
	var m map[string]any
	if err := json.Unmarshal(secs.AWS, &m); err != nil {
		t.Fatalf("aws section is not valid JSON object: %v", err)
	}
}

func TestLoad_DecoupledFromConnectors(t *testing.T) {
	// A sanity check that sections decode independently. We do not import the
	// connector packages here on purpose — that decoupling is the whole point.
	secs, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(secs.AWS) == 0 {
		t.Fatal("expected a non-empty raw aws section")
	}

	type awsShape struct {
		SSOStartURL   string `json:"sso_start_url"`
		SSORegion     string `json:"sso_region"`
		DefaultRegion string `json:"default_region"`
		PreferredRole string `json:"preferred_role,omitempty"`
	}

	var keys map[string]json.RawMessage
	if err := json.Unmarshal(secs.AWS, &keys); err != nil {
		t.Fatalf("aws section is not a JSON object: %v", err)
	}
	for _, key := range []string{"sso_start_url", "sso_region", "default_region"} {
		if _, ok := keys[key]; !ok {
			t.Fatalf("aws section missing required key %q", key)
		}
	}

	var a awsShape
	if err := json.Unmarshal(secs.AWS, &a); err != nil {
		t.Fatalf("aws section shape mismatch: %v", err)
	}
}
