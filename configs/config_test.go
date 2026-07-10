package configs

import (
	"encoding/json"
	"testing"
)

func TestLoad_ReturnsNamedSections(t *testing.T) {
	secs, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	// The committed config.json ships an identity.entra section and aws/azure/gcp
	// provider sections — present and valid JSON, without this package knowing
	// their schemas.
	if raw := secs.IdentitySection("entra"); len(raw) == 0 {
		t.Error("expected an identity.entra section")
	} else {
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Errorf("entra section not a JSON object: %v", err)
		}
	}
	for _, id := range []string{"aws", "azure", "gcp"} {
		if raw := secs.ProviderSection(id); len(raw) == 0 {
			t.Errorf("expected a providers.%s section", id)
		}
	}
}

func TestProviderSection_AbsentIsNil(t *testing.T) {
	secs := Sections{Providers: map[string]json.RawMessage{"aws": json.RawMessage(`{}`)}}
	if secs.ProviderSection("does-not-exist") != nil {
		t.Error("absent section should be nil")
	}
}