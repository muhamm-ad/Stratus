package aws

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
)

// --- config --------------------------------------------------------------

func TestParseConfig_FromSection(t *testing.T) {
	raw := json.RawMessage(`{
		"sso_start_url": "https://acme.awsapps.com/start",
		"sso_region": "eu-west-1",
		"default_region": "eu-west-3",
		"preferred_role": "StratusAdminRole"
	}`)
	c, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.SSOStartURL != "https://acme.awsapps.com/start" {
		t.Errorf("start url not parsed: %q", c.SSOStartURL)
	}
	if c.SSORegion != "eu-west-1" || c.DefaultRegion != "eu-west-3" {
		t.Errorf("regions not parsed: %+v", c)
	}
	if c.PreferredRole != "StratusAdminRole" {
		t.Errorf("role not parsed: %q", c.PreferredRole)
	}
}

func TestParseConfig_EnvOverridesSection(t *testing.T) {
	raw := json.RawMessage(`{"sso_start_url":"https://baked.example/start","sso_region":"us-east-1","default_region":"us-east-1"}`)
	// Env wins over the embedded section (dev/CI override).
	t.Setenv(EnvStartURL, "https://override.awsapps.com/start")
	t.Setenv(EnvDefaultRegion, "eu-west-1")

	c, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.SSOStartURL != "https://override.awsapps.com/start" {
		t.Errorf("env did not override start url: %q", c.SSOStartURL)
	}
	if c.DefaultRegion != "eu-west-1" {
		t.Errorf("env did not override default region: %q", c.DefaultRegion)
	}
	if c.SSORegion != "us-east-1" {
		t.Errorf("section value should remain when no env override: %q", c.SSORegion)
	}
}

func TestParseConfig_MissingFieldsError(t *testing.T) {
	// Empty section, no env -> validation must fail.
	t.Setenv(EnvStartURL, "")
	t.Setenv(EnvSSORegion, "")
	t.Setenv(EnvDefaultRegion, "")

	if _, err := ParseConfig(json.RawMessage(`{}`)); err == nil {
		t.Fatal("expected validation error for empty section")
	}
	// Nil section behaves like empty.
	if _, err := ParseConfig(nil); err == nil {
		t.Fatal("expected validation error for nil section")
	}
}

// --- pkce ----------------------------------------------------------------

func TestNewPKCE_ChallengeIsS256OfVerifier(t *testing.T) {
	p, err := newPKCE()
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(p.Verifier))
	want := base64.RawURLEncoding.EncodeToString(sum[:])
	if p.Challenge != want {
		t.Errorf("challenge mismatch:\n got %q\nwant %q", p.Challenge, want)
	}
	if len(p.Verifier) < 43 || len(p.Verifier) > 128 {
		t.Errorf("verifier length %d outside RFC 7636 bounds", len(p.Verifier))
	}
}

func TestNewPKCE_Unique(t *testing.T) {
	a, _ := newPKCE()
	b, _ := newPKCE()
	if a.Verifier == b.Verifier {
		t.Error("verifiers should be unique")
	}
}

// --- loopback ------------------------------------------------------------

func TestWaitForCode_HappyPath(t *testing.T) {
	ln, redirectURI, err := listenLoopback()
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		// Simulate the browser redirect from the authorization server.
		time.Sleep(20 * time.Millisecond)
		http.Get(redirectURI + "?state=xyz&code=the-code") //nolint:errcheck
	}()

	code, err := waitForCode(context.Background(), ln, "xyz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != "the-code" {
		t.Errorf("got code %q, want %q", code, "the-code")
	}
}

func TestWaitForCode_StateMismatch(t *testing.T) {
	ln, redirectURI, err := listenLoopback()
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		time.Sleep(20 * time.Millisecond)
		http.Get(redirectURI + "?state=WRONG&code=x") //nolint:errcheck
	}()

	_, err = waitForCode(context.Background(), ln, "xyz")
	if err == nil {
		t.Fatal("expected state-mismatch error")
	}
}

func TestWaitForCode_ContextCancel(t *testing.T) {
	ln, _, err := listenLoopback()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := waitForCode(ctx, ln, "xyz"); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

// --- full auth-code login with a mock OIDC client -------------------------

// mockOIDC simulates IAM Identity Center: RegisterClient returns an authorize
// endpoint pointing at an in-process test server that immediately redirects
// back to the loopback URI with a code, and CreateToken returns a session.
type mockOIDC struct {
	authServer *http.Server
	state      string
}

func (m *mockOIDC) RegisterClient(_ context.Context, in *ssooidc.RegisterClientInput, _ ...func(*ssooidc.Options)) (*ssooidc.RegisterClientOutput, error) {
	return &ssooidc.RegisterClientOutput{
		ClientId:              aws.String("client-id"),
		ClientSecret:          aws.String("client-secret"),
		AuthorizationEndpoint: aws.String("http://authorize.invalid/authorize"),
		TokenEndpoint:         aws.String("http://token.invalid/token"),
	}, nil
}

func (m *mockOIDC) StartDeviceAuthorization(context.Context, *ssooidc.StartDeviceAuthorizationInput, ...func(*ssooidc.Options)) (*ssooidc.StartDeviceAuthorizationOutput, error) {
	return nil, errors.New("not used in this test")
}

func (m *mockOIDC) CreateToken(_ context.Context, in *ssooidc.CreateTokenInput, _ ...func(*ssooidc.Options)) (*ssooidc.CreateTokenOutput, error) {
	switch aws.ToString(in.GrantType) {
	case grantAuthCode:
		if aws.ToString(in.Code) == "" || aws.ToString(in.CodeVerifier) == "" {
			return nil, errors.New("missing code or verifier")
		}
		return &ssooidc.CreateTokenOutput{
			AccessToken:  aws.String("access-token"),
			RefreshToken: aws.String("refresh-token"),
			ExpiresIn:    3600,
		}, nil
	case grantRefresh:
		if aws.ToString(in.RefreshToken) == "" {
			return nil, errors.New("missing refresh token")
		}
		return &ssooidc.CreateTokenOutput{
			AccessToken:  aws.String("access-token"),
			RefreshToken: aws.String("refresh-token-2"),
			ExpiresIn:    3600,
		}, nil
	default:
		return nil, errors.New("unexpected grant type: " + aws.ToString(in.GrantType))
	}
}

func TestLogin_AuthCodeFlow_EndToEnd(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	store := &memoryStore{}
	mock := &mockOIDC{}

	p := &Provider{
		cfg:   Config{SSOStartURL: "https://acme.awsapps.com/start", SSORegion: "eu-west-1", DefaultRegion: "eu-west-1"},
		oidc:  mock,
		store: store,
		now:   func() time.Time { return now },
		// Instead of opening a real browser, parse the authorize URL and
		// perform the redirect back to the loopback server ourselves.
		openBrowser: func(rawURL string) error {
			u, err := parseAuthorizeURL(rawURL)
			if err != nil {
				return err
			}
			go func() {
				http.Get(u.RedirectURI + "?state=" + u.State + "&code=auth-code") //nolint:errcheck
			}()
			return nil
		},
	}

	if err := p.Login(context.Background()); err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if !p.IsAuthenticated() {
		t.Fatal("provider should be authenticated after login")
	}
	saved, _ := store.Load()
	if saved == nil || saved.AccessToken != "access-token" {
		t.Fatalf("token not persisted: %+v", saved)
	}
	if saved.RefreshToken != "refresh-token" {
		t.Error("refresh token not stored")
	}
	wantExpiry := now.Add(time.Hour)
	if !saved.ExpiresAt.Equal(wantExpiry) {
		t.Errorf("expiry %v, want %v", saved.ExpiresAt, wantExpiry)
	}
}

func TestValidToken_RefreshesWhenExpiringSoon(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	p := &Provider{
		cfg:   Config{SSORegion: "eu-west-1"},
		oidc:  &mockOIDC{},
		store: &memoryStore{},
		now:   func() time.Time { return now },
		token: &Token{
			AccessToken:  "old",
			RefreshToken: "refresh-token",
			ExpiresAt:    now.Add(2 * time.Minute), // within refreshWindow
			ClientID:     "client-id",
			ClientSecret: "client-secret",
		},
	}
	tok, err := p.validToken(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok.AccessToken != "access-token" {
		t.Errorf("expected refreshed token, got %q", tok.AccessToken)
	}
}

// parseAuthorizeURL is a tiny test helper extracting the fields the mock needs.
type authorizeURL struct {
	RedirectURI string
	State       string
}

func parseAuthorizeURL(raw string) (authorizeURL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return authorizeURL{}, err
	}
	q := u.Query()
	return authorizeURL{RedirectURI: q.Get("redirect_uri"), State: q.Get("state")}, nil
}
