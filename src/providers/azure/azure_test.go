package azure

import (
	"context"
	"testing"

	"github.com/muhamm-ad/stratus/core"
)

type fakeIDP struct {
	armToken string
	gotScope string
}

func (f *fakeIDP) Login(context.Context) error             { return nil }
func (f *fakeIDP) IsAuthenticated() bool                   { return true }
func (f *fakeIDP) IDToken(context.Context) (string, error) { return "", nil }
func (f *fakeIDP) AccessToken(_ context.Context, scope string) (string, error) {
	f.gotScope = scope
	return f.armToken, nil
}
func (f *fakeIDP) Logout(context.Context) error { return nil }

func TestAuthenticate_AcquiresARMToken(t *testing.T) {
	p := New(Config{SubscriptionID: "sub-123"})
	idp := &fakeIDP{armToken: "ARM"}

	if err := p.Authenticate(context.Background(), idp); err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if p.AccessToken() != "ARM" || !p.IsAuthenticated() {
		t.Errorf("token not stored")
	}
	if idp.gotScope != "https://management.azure.com/.default" {
		t.Errorf("requested scope = %q", idp.gotScope)
	}
}

var _ core.ProviderConnector = (*Provider)(nil)
