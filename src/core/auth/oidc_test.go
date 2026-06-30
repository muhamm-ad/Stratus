package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

// tokenServer fakes an OIDC token endpoint.
func tokenServer(t *testing.T, wantGrant string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if g := r.Form.Get("grant_type"); g != wantGrant {
			http.Error(w, "bad grant: "+g, http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"AT","id_token":"IDT","refresh_token":"RT","expires_in":3600,"scope":"openid"}`))
	}))
}

func TestLoginAuthCode_EndToEnd(t *testing.T) {
	srv := tokenServer(t, "authorization_code")
	defer srv.Close()
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	cfg := OIDCConfig{
		ClientID:          "cid",
		AuthorizeEndpoint: "http://authorize.invalid/authorize",
		TokenEndpoint:     srv.URL,
		Scope:             "openid offline_access",
		HTTPClient:        srv.Client(),
		Now:               func() time.Time { return now },
	}
	// Fake browser: parse the authorize URL and hit the loopback with a code.
	open := func(raw string) error {
		u, err := url.Parse(raw)
		if err != nil {
			return err
		}
		q := u.Query()
		go func() {
			http.Get(q.Get("redirect_uri") + "?state=" + q.Get("state") + "&code=abc") //nolint:errcheck
		}()
		return nil
	}

	tok, err := LoginAuthCode(context.Background(), cfg, open)
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if tok.IDToken != "IDT" || tok.RefreshToken != "RT" || tok.AccessToken != "AT" {
		t.Errorf("unexpected token: %+v", tok)
	}
	if !tok.ExpiresAt.Equal(now.Add(time.Hour)) {
		t.Errorf("expiry = %v", tok.ExpiresAt)
	}
}

func TestRefreshToken(t *testing.T) {
	srv := tokenServer(t, "refresh_token")
	defer srv.Close()
	cfg := OIDCConfig{ClientID: "cid", TokenEndpoint: srv.URL, Scope: "https://management.azure.com/.default", HTTPClient: srv.Client()}
	tok, err := RefreshToken(context.Background(), cfg, "RT")
	if err != nil {
		t.Fatal(err)
	}
	if tok.AccessToken != "AT" {
		t.Errorf("got %q", tok.AccessToken)
	}
}