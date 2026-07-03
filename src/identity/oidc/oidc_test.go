package oidc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/muhamm-ad/stratus/core/auth"
)

func redirectFromAuthorizeURL(t *testing.T, rawurl string) string {
	t.Helper()
	u, err := url.Parse(rawurl)
	if err != nil {
		t.Fatalf("parse authorize url: %v", err)
	}
	q := u.Query()
	redirect := q.Get("redirect_uri")
	state := q.Get("state")
	// Simulate the IdP redirecting back with an auth code.
	return redirect + "?code=fake-code&state=" + url.QueryEscape(state)
}

func TestParseConfig(t *testing.T) {
	// issuer-only is valid (endpoints discovered later)
	if _, err := ParseConfig(json.RawMessage(`{"issuer":"https://idp.example/","client_id":"abc"}`)); err != nil {
		t.Fatalf("issuer form should be valid: %v", err)
	}
	// endpoints-only is valid
	if _, err := ParseConfig(json.RawMessage(`{"authorize_endpoint":"https://a","token_endpoint":"https://t","client_id":"abc"}`)); err != nil {
		t.Fatalf("endpoint form should be valid: %v", err)
	}
	// missing client_id fails
	if _, err := ParseConfig(json.RawMessage(`{"issuer":"https://idp.example/"}`)); err == nil {
		t.Fatal("missing client_id should fail")
	}
	// neither issuer nor endpoints fails
	if _, err := ParseConfig(json.RawMessage(`{"client_id":"abc"}`)); err == nil {
		t.Fatal("missing issuer/endpoints should fail")
	}
	// default scopes
	c, _ := ParseConfig(json.RawMessage(`{"issuer":"https://idp.example/","client_id":"abc"}`))
	if c.scopeString() != "openid offline_access" {
		t.Errorf("unexpected default scopes: %q", c.scopeString())
	}
}

// TestLogin_WithDiscovery drives a full login against a fake OIDC provider whose
// endpoints are found via the discovery document.
func TestLogin_WithDiscovery(t *testing.T) {
	mux := http.NewServeMux()
	var srv *httptest.Server

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"authorization_endpoint": srv.URL + "/authorize",
			"token_endpoint":         srv.URL + "/token",
		})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "AT", "id_token": "IDT", "refresh_token": "RT", "expires_in": 3600,
		})
	})
	srv = httptest.NewServer(mux)
	defer srv.Close()

	// Fake browser: hit the loopback redirect the authorize URL points to.
	open := func(rawurl string) error {
		go func() { _, _ = http.Get(redirectFromAuthorizeURL(t, rawurl)) }()
		return nil
	}

	p := New("fake",
		Config{Issuer: srv.URL, ClientID: "cid"},
		WithTokenStore(&auth.MemoryStore{}),
		WithBrowserOpener(open),
		WithHTTPClient(srv.Client()),
		WithClock(time.Now),
	)

	if err := p.Login(context.Background()); err != nil {
		t.Fatalf("login: %v", err)
	}
	if !p.IsAuthenticated() {
		t.Fatal("should be authenticated after login")
	}
	idt, err := p.IDToken(context.Background())
	if err != nil || idt != "IDT" {
		t.Fatalf("id_token: %q err=%v", idt, err)
	}
}

func TestAccessToken_UsesRefreshToken(t *testing.T) {
	now := time.Now()
	var srv *httptest.Server
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("grant_type") != "refresh_token" {
			http.Error(w, "want refresh_token", http.StatusBadRequest)
			return
		}
		// echo the requested scope back as a new rotated refresh token
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "AT-" + r.Form.Get("scope"), "refresh_token": "RT2", "expires_in": 3600,
		})
	})
	srv = httptest.NewServer(mux)
	defer srv.Close()

	// Seed a logged-in session via the store (no login round-trip needed).
	store := &auth.MemoryStore{}
	_ = store.Save(&auth.TokenResponse{IDToken: "IDT", RefreshToken: "RT", ExpiresAt: now.Add(time.Hour)})

	p := New("fake",
		Config{AuthorizeEndpoint: srv.URL + "/authorize", TokenEndpoint: srv.URL + "/token", ClientID: "cid"},
		WithTokenStore(store), WithHTTPClient(srv.Client()), WithClock(func() time.Time { return now }),
	)

	tok, err := p.AccessToken(context.Background(), "https://management.azure.com/.default")
	if err != nil {
		t.Fatalf("access token: %v", err)
	}
	if tok != "AT-https://management.azure.com/.default" {
		t.Errorf("unexpected access token: %q", tok)
	}
	// refresh token should have rotated in the store
	saved, _ := store.Load()
	if saved.RefreshToken != "RT2" {
		t.Errorf("refresh token not rotated: %q", saved.RefreshToken)
	}
}
