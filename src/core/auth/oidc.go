package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// OIDCConfig describes a generic OIDC client (e.g. the Stratus Entra app).
type OIDCConfig struct {
	ClientID          string
	AuthorizeEndpoint string
	TokenEndpoint     string
	Scope             string

	// HTTPClient and Now are optional; sensible defaults are used when nil.
	HTTPClient *http.Client
	Now        func() time.Time
}

func (c OIDCConfig) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

func (c OIDCConfig) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

// LoginAuthCode runs the Authorization Code + PKCE flow over a loopback
// redirect (RFC 8252): it opens the browser once and exchanges the returned
// code for tokens. `open` is the browser opener (injected in tests).
func LoginAuthCode(ctx context.Context, cfg OIDCConfig, open func(string) error) (*TokenResponse, error) {
	ln, redirectURI, err := listenLoopback()
	if err != nil {
		return nil, err
	}
	pkce, err := newPKCE()
	if err != nil {
		return nil, err
	}
	state, err := newState()
	if err != nil {
		return nil, err
	}

	authURL := buildAuthorizeURL(cfg.AuthorizeEndpoint, authorizeParams{
		ClientID: cfg.ClientID, RedirectURI: redirectURI, State: state,
		Challenge: pkce.Challenge, Scope: cfg.Scope,
	})
	if err := open(authURL); err != nil {
		_ = ln.Close()
		return nil, fmt.Errorf("auth: open browser: %w", err)
	}

	code, err := waitForCode(ctx, ln, state)
	if err != nil {
		return nil, err
	}

	return postToken(ctx, cfg, url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"code_verifier": {pkce.Verifier},
		"client_id":     {cfg.ClientID},
		"redirect_uri":  {redirectURI},
		"scope":         {cfg.Scope},
	})
}

// RefreshToken redeems a refresh token for a fresh token set, optionally for a
// different scope (used to acquire e.g. an ARM-scoped token silently).
func RefreshToken(ctx context.Context, cfg OIDCConfig, refreshToken string) (*TokenResponse, error) {
	return postToken(ctx, cfg, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {cfg.ClientID},
		"scope":         {cfg.Scope},
	})
}

func postToken(ctx context.Context, cfg OIDCConfig, form url.Values) (*TokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := cfg.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("auth: token request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("auth: token endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var raw struct {
		AccessToken  string `json:"access_token"`
		IDToken      string `json:"id_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("auth: decode token response: %w", err)
	}
	return &TokenResponse{
		AccessToken:  raw.AccessToken,
		IDToken:      raw.IDToken,
		RefreshToken: raw.RefreshToken,
		ExpiresAt:    cfg.now().Add(time.Duration(raw.ExpiresIn) * time.Second),
		Scope:        raw.Scope,
	}, nil
}

type authorizeParams struct {
	ClientID    string
	RedirectURI string
	State       string
	Challenge   string
	Scope       string
}

func buildAuthorizeURL(endpoint string, p authorizeParams) string {
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", p.ClientID)
	q.Set("redirect_uri", p.RedirectURI)
	q.Set("state", p.State)
	q.Set("code_challenge", p.Challenge)
	q.Set("code_challenge_method", "S256")
	if p.Scope != "" {
		q.Set("scope", p.Scope)
	}
	return endpoint + "?" + q.Encode()
}