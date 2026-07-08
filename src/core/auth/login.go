package auth

import (
	"context"
	// "fmt"
	// "os"
	"time"

	"github.com/int128/oauth2cli"
	"github.com/pkg/browser"
	"golang.org/x/oauth2"

	"github.com/muhamm-ad/stratus/core"
)

// BrowserLogin runs the RFC 8252 flow. oauth2cli spins up a loopback server on
// 127.0.0.1 and BLOCKS in GetToken until the callback arrives — but it does NOT
// open the browser itself. We listen on its ready channel and open the system
// browser (with a printed fallback URL). PKCE (S256) is added via x/oauth2.
func (c *Client) BrowserLogin(ctx context.Context) (*oauth2.Token, error) {
	// Local context so the opener goroutine is torn down when GetToken returns.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	verifier := oauth2.GenerateVerifier()
	ready := make(chan string, 1) // buffered: never block oauth2cli's send

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
		LocalServerReadyChan:   ready,
	}

	go func() {
		select {
		case url := <-ready:
			// fmt.Fprintf(os.Stderr, "\nOpening your browser to sign in.\nIf it does not open, visit:\n  %s\n\n", url)
			_ = browser.OpenURL(url)
		case <-ctx.Done():
		}
	}()

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
