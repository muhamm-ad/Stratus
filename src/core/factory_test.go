package core

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

// stubConnector is a no-op ProviderConnector for factory tests.
type stubConnector struct{ id ProviderID }

func (s stubConnector) ID() ProviderID                                            { return s.id }
func (s stubConnector) Authenticate(context.Context, IdentityProvider) error      { return nil }
func (s stubConnector) IsAuthenticated() bool                                     { return false }
func (s stubConnector) ListInstances(context.Context, string) ([]Instance, error) { return nil, nil }
func (s stubConnector) Connect(context.Context, ConnectRequest) (Session, error)  { return nil, nil }
func (s stubConnector) Logout(context.Context) error                              { return nil }

func TestRegisterProvider_AndFactories(t *testing.T) {
	// Use a unique ID so this test does not depend on which provider packages
	// happen to be imported into the test binary.
	const id ProviderID = "test-factory-xyz"
	RegisterProvider(id, func(json.RawMessage) (ProviderConnector, error) {
		return stubConnector{id: id}, nil
	})
	if _, ok := Factories()[id]; !ok {
		t.Fatal("factory not present after RegisterProvider")
	}
}

func TestBuildAll_SkipsInvalidAndBuildsValid(t *testing.T) {
	good := ProviderID("good")
	bad := ProviderID("bad")
	facs := map[ProviderID]Factory{
		good: func(raw json.RawMessage) (ProviderConnector, error) { return stubConnector{id: good}, nil },
		bad:  func(raw json.RawMessage) (ProviderConnector, error) { return nil, errors.New("incomplete config") },
	}
	sections := map[string]json.RawMessage{
		"good": json.RawMessage(`{}`),
		"bad":  json.RawMessage(`{}`),
	}
	reg, errs := BuildAll(facs, sections)

	if _, ok := reg.Get(good); !ok {
		t.Error("good provider should be registered")
	}
	if _, ok := reg.Get(bad); ok {
		t.Error("bad provider must not be registered")
	}
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
}
