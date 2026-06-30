package gcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type fakeIDP struct{ idToken string }

func (f fakeIDP) Login(context.Context) error                         { return nil }
func (f fakeIDP) IsAuthenticated() bool                               { return true }
func (f fakeIDP) IDToken(context.Context) (string, error)             { return f.idToken, nil }
func (f fakeIDP) AccessToken(context.Context, string) (string, error) { return "", nil }
func (f fakeIDP) Logout(context.Context) error                        { return nil }

func TestAuthenticate_TokenExchange(t *testing.T) {
	var gotSubject, gotGrant string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotGrant = r.Form.Get("grant_type")
		gotSubject = r.Form.Get("subject_token")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"GCP_AT","expires_in":3600}`))
	}))
	defer srv.Close()
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	p := New(GCPConfig{WorkforceAudience: "//iam.googleapis.com/x", Scope: DefaultScope}, WithHTTPClient(srv.Client()), WithSTSEndpoint(srv.URL), WithClock(func() time.Time { return now }))

	if err := p.Authenticate(context.Background(), fakeIDP{idToken: "ENTRA_JWT"}); err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if gotGrant != "urn:ietf:params:oauth:grant-type:token-exchange" {
		t.Errorf("grant = %q", gotGrant)
	}
	if gotSubject != "ENTRA_JWT" {
		t.Errorf("subject = %q", gotSubject)
	}
	if p.AccessToken() != "GCP_AT" || !p.IsAuthenticated() {
		t.Errorf("token not stored: %q", p.AccessToken())
	}
}
