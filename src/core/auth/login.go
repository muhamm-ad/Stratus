package auth

import (
	"context"
	"time"

	"github.com/int128/oauth2cli"
	"golang.org/x/oauth2"

	"github.com/muhamm-ad/stratus/core"
)

// BrowserLogin runs the RFC 8252 flow: oauth2cli spins up a loopback server on
// 127.0.0.1, opens the system browser, handles the state check, and we add PKCE
// (S256) via x/oauth2's built-in verifier helpers.
func (c *Client) BrowserLogin(ctx context.Context) (*oauth2.Token, error) {
	verifier := oauth2.GenerateVerifier()
	cfg := oauth2cli.Config{
		OAuth2Config: c.OAuth,
		AuthCodeOptions: []oauth2.AuthCodeOption{
			oauth2.AccessTypeOffline, // request a refresh token
			oauth2.S256ChallengeOption(verifier),
		},
		TokenRequestOptions: []oauth2.AuthCodeOption{
			oauth2.VerifierOption(verifier),
		},
		LocalServerBindAddress: []string{"127.0.0.1:0"}, // ephemeral loopback port
	}
	return oauth2cli.GetToken(ctx, cfg)
}

// DeviceLogin runs the RFC 8628 device-authorization flow (headless fallback).
// The code is surfaced through onCode; DeviceAccessToken polls until completion.
func (c *Client) DeviceLogin(ctx context.Context, onCode func(core.DeviceCode)) (*oauth2.Token, error) {
	da, err := c.OAuth.DeviceAuth(ctx)
	if err != nil {
		return nil, err
	}
	if onCode != nil {
		onCode(core.DeviceCode{
			UserCode:        da.UserCode,
			VerificationURI: da.VerificationURI,
			Interval:        time.Duration(da.Interval) * time.Second,
		})
	}
	return c.OAuth.DeviceAccessToken(ctx, da)
}