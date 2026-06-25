package core

import (
	"context"
	"testing"
)

// fakeConnector is a minimal CloudConnector for registry tests.
type fakeConnector struct{ id ProviderID }

func (f fakeConnector) ID() ProviderID              { return f.id }
func (f fakeConnector) Login(context.Context) error { return nil }
func (f fakeConnector) IsAuthenticated() bool       { return false }
func (f fakeConnector) ListAccounts(context.Context) ([]Account, error) {
	return nil, ErrNotImplemented
}
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
	if !ok {
		t.Fatal("expected AWS connector to be registered")
	}
	if got.ID() != ProviderAWS {
		t.Errorf("got id %q, want %q", got.ID(), ProviderAWS)
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
	if len(ids) != len(want) {
		t.Fatalf("got %d ids, want %d", len(ids), len(want))
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Errorf("ids[%d] = %q, want %q", i, ids[i], want[i])
		}
	}
}
