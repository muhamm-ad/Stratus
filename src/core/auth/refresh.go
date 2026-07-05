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

	"golang.org/x/oauth2"
)

// RefreshGrant performs an OAuth2 refresh_token grant with an EXPLICIT scope.
//
// This is the ONE deliberately hand-written OAuth call in Stratus. Deriving an
// Azure ARM token from the existing Entra session requires sending
// scope=https://management.azure.com/.default ON THE REFRESH REQUEST (Entra
// re-scopes a refresh token to another resource). golang.org/x/oauth2
// deliberately never sends a scope on refresh, and MSAL cannot import a refresh
// token obtained outside it — so this small call is the documented gap. Every
// other token operation goes through x/oauth2 / go-oidc.
func RefreshGrant(ctx context.Context, hc *http.Client, tokenURL, clientID, refreshToken string, scopes []string) (*oauth2.Token, error) {
	if hc == nil {
		hc = http.DefaultClient
	}
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {clientID},
		"refresh_token": {refreshToken},
	}
	if len(scopes) > 0 {
		form.Set("scope", strings.Join(scopes, " "))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh grant: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var r struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
		IDToken      string `json:"id_token"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("refresh grant: decode: %w", err)
	}
	tok := &oauth2.Token{AccessToken: r.AccessToken, TokenType: r.TokenType, RefreshToken: r.RefreshToken}
	if r.ExpiresIn > 0 {
		tok.Expiry = time.Now().Add(time.Duration(r.ExpiresIn) * time.Second)
	}
	if r.IDToken != "" {
		tok = tok.WithExtra(map[string]any{"id_token": r.IDToken})
	}
	return tok, nil
}
