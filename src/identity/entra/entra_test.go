package entra

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/muhamm-ad/stratus/core/auth"
)

func TestLogin_AndIDToken(t *testing.T) {
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"AT","id_token":"IDT","refresh_token":"RT","expires_in":3600}`))
	}))
	defer tokenSrv.Close()
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	p := New(Config{TenantID: "t", ClientID: "cid"},
		WithTokenStore(&auth.MemoryStore{}),
		WithHTTPClient(tokenSrv.Client()),
		WithClock(func() time.Time { return now }),
		WithEndpoints("http://authorize.invalid/authorize", tokenSrv.URL),
		WithBrowserOpener(func(raw string) error {
			u, _ := url.Parse(raw)
			q := u.Query()
			go func() { http.Get(q.Get("redirect_uri") + "?state=" + q.Get("state") + "&code=abc") }() //nolint:errcheck
			return nil
		}),
	)

	if err := p.Login(context.Background()); err != nil {
		t.Fatalf("login: %v", err)
	}
	if !p.IsAuthenticated() {
		t.Fatal("should be authenticated")
	}
	idt, err := p.IDToken(context.Background())
	if err != nil || idt != "IDT" {
		t.Fatalf("IDToken = (%q,%v)", idt, err)
	}
}

func TestAccessToken_UsesRefreshToken(t *testing.T) {
	var gotGrant, gotScope string
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotGrant = r.Form.Get("grant_type")
		gotScope = r.Form.Get("scope")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"ARM_TOKEN","refresh_token":"RT2","expires_in":3600}`))
	}))
	defer tokenSrv.Close()
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	p := New(Config{TenantID: "t", ClientID: "cid"},
		WithTokenStore(&auth.MemoryStore{}),
		WithHTTPClient(tokenSrv.Client()),
		WithClock(func() time.Time { return now }),
		WithEndpoints("http://authorize.invalid", tokenSrv.URL),
	)
	// Seed a session as if already logged in.
	p.tok = &auth.TokenResponse{IDToken: "IDT", RefreshToken: "RT", ExpiresAt: now.Add(time.Hour)}

	tok, err := p.AccessToken(context.Background(), "https://management.azure.com/.default")
	if err != nil {
		t.Fatal(err)
	}
	if tok != "ARM_TOKEN" {
		t.Errorf("token = %q", tok)
	}
	if gotGrant != "refresh_token" {
		t.Errorf("grant = %q", gotGrant)
	}
	if gotScope != "https://management.azure.com/.default" {
		t.Errorf("scope = %q", gotScope)
	}
	if p.tok.RefreshToken != "RT2" {
		t.Errorf("refresh token not rotated: %q", p.tok.RefreshToken)
	}
}
