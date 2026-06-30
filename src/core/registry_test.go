package core

import (
	"context"
	"testing"
)

// fakeConnector is a minimal ProviderConnector for registry tests.
type fakeConnector struct{ id ProviderID }

func (f fakeConnector) ID() ProviderID                                       { return f.id }
func (f fakeConnector) Authenticate(context.Context, IdentityProvider) error { return nil }
func (f fakeConnector) IsAuthenticated() bool                                { return false }
func (f fakeConnector) ListInstances(context.Context, string) ([]Instance, error) {
	return nil, ErrNotImplemented
}
func (f fakeConnector) Connect(context.Context, ConnectRequest) (Session, error) {
	return nil, ErrNotImplemented
}
func (f fakeConnector) Logout(context.Context) error { return nil }

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := NewRegistry()
	r.Register(fakeConnector{id: ProviderAWS})
	got, ok := r.Get(ProviderAWS)
	if !ok || got.ID() != ProviderAWS {
		t.Fatalf("AWS connector not registered correctly")
	}
	if _, ok := r.Get(ProviderGCP); ok {
		t.Error("did not expect a GCP connector")
	}
}

func TestRegistry_IDsSorted(t *testing.T) {
	r := NewRegistry()
	r.Register(fakeConnector{id: ProviderGCP})
	r.Register(fakeConnector{id: ProviderAWS})
	r.Register(fakeConnector{id: ProviderAzure})
	ids := r.IDs()
	want := []ProviderID{ProviderAWS, ProviderAzure, ProviderGCP}
	for i := range want {
		if ids[i] != want[i] {
			t.Errorf("ids[%d] = %q, want %q", i, ids[i], want[i])
		}
	}
}
