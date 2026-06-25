package aws

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	ssotypes "github.com/aws/aws-sdk-go-v2/service/ssooidc/types"

	"github.com/muhamm-ad/stratus/core"
)

// OIDCClient is the subset of *ssooidc.Client used by the auth flows. Defining
// it as an interface lets tests inject a mock without touching the network.
type OIDCClient interface {
	RegisterClient(context.Context, *ssooidc.RegisterClientInput, ...func(*ssooidc.Options)) (*ssooidc.RegisterClientOutput, error)
	StartDeviceAuthorization(context.Context, *ssooidc.StartDeviceAuthorizationInput, ...func(*ssooidc.Options)) (*ssooidc.StartDeviceAuthorizationOutput, error)
	CreateToken(context.Context, *ssooidc.CreateTokenInput, ...func(*ssooidc.Options)) (*ssooidc.CreateTokenOutput, error)
}

const (
	grantAuthCode = "authorization_code"
	grantDevice   = "urn:ietf:params:oauth:grant-type:device_code"
	grantRefresh  = "refresh_token"
	ssoScope      = "sso:account:access"
)

// authenticate runs the primary RFC 8252 flow and, if a loopback listener
// cannot be bound, transparently falls back to the RFC 8628 device flow.
func (p *Provider) authenticate(ctx context.Context) (*Token, error) {
	tok, err := p.loginAuthCode(ctx)
	if errors.Is(err, errLoopbackUnavailable) {
		return p.loginDeviceCode(ctx)
	}
	return tok, err
}

// loginAuthCode implements OAuth 2.0 Authorization Code + PKCE over a loopback
// redirect (RFC 8252) — the "open the browser and come back automatically"
// experience.
func (p *Provider) loginAuthCode(ctx context.Context) (*Token, error) {
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

	reg, err := p.oidc.RegisterClient(ctx, &ssooidc.RegisterClientInput{
		ClientName:   aws.String("stratus"),
		ClientType:   aws.String("public"),
		Scopes:       []string{ssoScope},
		GrantTypes:   []string{grantAuthCode, grantRefresh},
		RedirectUris: []string{redirectURI},
		IssuerUrl:    aws.String(p.cfg.SSOStartURL),
	})
	if err != nil {
		_ = ln.Close()
		return nil, fmt.Errorf("aws: register client: %w", err)
	}

	authURL := buildAuthorizeURL(aws.ToString(reg.AuthorizationEndpoint), authorizeParams{
		ClientID:    aws.ToString(reg.ClientId),
		RedirectURI: redirectURI,
		State:       state,
		Challenge:   pkce.Challenge,
		Scope:       ssoScope,
	})

	if err := p.openBrowser(authURL); err != nil {
		// Not fatal: the user can open the printed URL manually. Surface it via
		// the device-code handler channel if one is configured.
		if p.onDeviceCode != nil {
			p.onDeviceCode(authURL, "")
		}
	}

	code, err := waitForCode(ctx, ln, state)
	if err != nil {
		return nil, err
	}

	out, err := p.oidc.CreateToken(ctx, &ssooidc.CreateTokenInput{
		ClientId:     reg.ClientId,
		ClientSecret: reg.ClientSecret,
		GrantType:    aws.String(grantAuthCode),
		Code:         aws.String(code),
		CodeVerifier: aws.String(pkce.Verifier),
		RedirectUri:  aws.String(redirectURI),
	})
	if err != nil {
		return nil, fmt.Errorf("aws: exchange code for token: %w", err)
	}

	return p.tokenFrom(reg, out), nil
}

// loginDeviceCode implements the OAuth 2.0 Device Authorization Grant
// (RFC 8628), used when no loopback/browser is available. This mirrors the
// behaviour of the legacy Python tool, without the embedded-webview hack.
func (p *Provider) loginDeviceCode(ctx context.Context) (*Token, error) {
	// A minimal registration (no redirect URIs) defaults to device auth.
	reg, err := p.oidc.RegisterClient(ctx, &ssooidc.RegisterClientInput{
		ClientName: aws.String("stratus"),
		ClientType: aws.String("public"),
		Scopes:     []string{ssoScope},
	})
	if err != nil {
		return nil, fmt.Errorf("aws: register client: %w", err)
	}

	da, err := p.oidc.StartDeviceAuthorization(ctx, &ssooidc.StartDeviceAuthorizationInput{
		ClientId:     reg.ClientId,
		ClientSecret: reg.ClientSecret,
		StartUrl:     aws.String(p.cfg.SSOStartURL),
	})
	if err != nil {
		return nil, fmt.Errorf("aws: start device authorization: %w", err)
	}

	// Show the verification URL + user code, and try to open the browser.
	if p.onDeviceCode != nil {
		p.onDeviceCode(aws.ToString(da.VerificationUriComplete), aws.ToString(da.UserCode))
	}
	_ = p.openBrowser(aws.ToString(da.VerificationUriComplete))

	interval := time.Duration(maxInt(int(da.Interval), 1)) * time.Second
	deadline := p.now().Add(time.Duration(maxInt(int(da.ExpiresIn), 60)) * time.Second)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}

		if p.now().After(deadline) {
			return nil, errors.New("aws: device authorization timed out")
		}

		out, err := p.oidc.CreateToken(ctx, &ssooidc.CreateTokenInput{
			ClientId:     reg.ClientId,
			ClientSecret: reg.ClientSecret,
			GrantType:    aws.String(grantDevice),
			DeviceCode:   da.DeviceCode,
		})
		if err == nil {
			return p.tokenFrom(reg, out), nil
		}

		var pending *ssotypes.AuthorizationPendingException
		var slow *ssotypes.SlowDownException
		switch {
		case errors.As(err, &pending):
			continue
		case errors.As(err, &slow):
			interval += 5 * time.Second
			continue
		default:
			return nil, fmt.Errorf("aws: device token poll: %w", err)
		}
	}
}

// refresh exchanges the stored refresh token for a fresh access token.
func (p *Provider) refresh(ctx context.Context, t *Token) (*Token, error) {
	if t == nil || t.RefreshToken == "" {
		return nil, core.ErrTokenExpired
	}
	out, err := p.oidc.CreateToken(ctx, &ssooidc.CreateTokenInput{
		ClientId:     aws.String(t.ClientID),
		ClientSecret: aws.String(t.ClientSecret),
		GrantType:    aws.String(grantRefresh),
		RefreshToken: aws.String(t.RefreshToken),
	})
	if err != nil {
		return nil, core.ErrTokenExpired
	}
	nt := &Token{
		AccessToken:  aws.ToString(out.AccessToken),
		RefreshToken: firstNonEmpty(aws.ToString(out.RefreshToken), t.RefreshToken),
		ExpiresAt:    p.now().Add(time.Duration(out.ExpiresIn) * time.Second),
		Region:       t.Region,
		StartURL:     t.StartURL,
		ClientID:     t.ClientID,
		ClientSecret: t.ClientSecret,
	}
	return nt, nil
}

// tokenFrom builds a Token from a registration + token response.
func (p *Provider) tokenFrom(reg *ssooidc.RegisterClientOutput, out *ssooidc.CreateTokenOutput) *Token {
	return &Token{
		AccessToken:  aws.ToString(out.AccessToken),
		RefreshToken: aws.ToString(out.RefreshToken),
		ExpiresAt:    p.now().Add(time.Duration(out.ExpiresIn) * time.Second),
		Region:       p.cfg.SSORegion,
		StartURL:     p.cfg.SSOStartURL,
		ClientID:     aws.ToString(reg.ClientId),
		ClientSecret: aws.ToString(reg.ClientSecret),
	}
}

type authorizeParams struct {
	ClientID    string
	RedirectURI string
	State       string
	Challenge   string
	Scope       string
}

// buildAuthorizeURL assembles the /authorize URL for the PKCE flow.
func buildAuthorizeURL(endpoint string, p authorizeParams) string {
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", p.ClientID)
	q.Set("redirect_uri", p.RedirectURI)
	q.Set("state", p.State)
	q.Set("code_challenge", p.Challenge)
	q.Set("code_challenge_method", "S256")
	if p.Scope != "" {
		q.Set("scopes", p.Scope)
	}
	return endpoint + "?" + q.Encode()
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
