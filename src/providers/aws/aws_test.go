package aws

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	ststypes "github.com/aws/aws-sdk-go-v2/service/sts/types"

	"github.com/muhamm-ad/stratus/core"
)

// fakeIDP implements core.IdentityProvider with canned tokens.
type fakeIDP struct {
	idToken string
	idErr   error
}

func (f fakeIDP) Login(context.Context) error { return nil }
func (f fakeIDP) IsAuthenticated() bool       { return true }
func (f fakeIDP) IDToken(context.Context) (string, error) {
	return f.idToken, f.idErr
}
func (f fakeIDP) AccessToken(context.Context, string) (string, error) { return "", nil }
func (f fakeIDP) Logout(context.Context) error                        { return nil }

// mockAssumer records the web identity token and returns canned creds.
type mockAssumer struct {
	gotToken string
	err      error
	exp      time.Time
}

func (m *mockAssumer) AssumeRoleWithWebIdentity(_ context.Context, in *sts.AssumeRoleWithWebIdentityInput, _ ...func(*sts.Options)) (*sts.AssumeRoleWithWebIdentityOutput, error) {
	m.gotToken = aws.ToString(in.WebIdentityToken)
	if m.err != nil {
		return nil, m.err
	}
	return &sts.AssumeRoleWithWebIdentityOutput{
		Credentials: &ststypes.Credentials{
			AccessKeyId:     aws.String("AKIA"),
			SecretAccessKey: aws.String("secret"),
			SessionToken:    aws.String("token"),
			Expiration:      aws.Time(m.exp),
		},
	}, nil
}

func TestAuthenticate_AssumeRoleWithWebIdentity(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	mock := &mockAssumer{exp: now.Add(time.Hour)}
	p := New(Config{RoleArn: "arn:aws:iam::123:role/x", Region: "eu-west-1"},
		WithAssumerFactory(func(string) assumer { return mock }),
		WithClock(func() time.Time { return now }),
	)
	if err := p.Authenticate(context.Background(), fakeIDP{idToken: "ENTRA_JWT"}); err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if mock.gotToken != "ENTRA_JWT" {
		t.Errorf("STS got token %q", mock.gotToken)
	}
	if !p.IsAuthenticated() {
		t.Error("should be authenticated")
	}
	if c := p.Credentials(); c == nil || aws.ToString(c.AccessKeyId) != "AKIA" {
		t.Errorf("creds not stored: %+v", c)
	}
}

func TestAuthenticate_ExchangeError(t *testing.T) {
	mock := &mockAssumer{err: errors.New("denied")}
	p := New(Config{RoleArn: "arn", Region: "eu-west-1"}, WithAssumerFactory(func(string) assumer { return mock }))
	err := p.Authenticate(context.Background(), fakeIDP{idToken: "x"})
	if !errors.Is(err, core.ErrExchange) {
		t.Fatalf("expected ErrExchange, got %v", err)
	}
}

func TestParseConfig_EnvOverride(t *testing.T) {
	env := map[string]string{EnvRoleArn: "arn:aws:iam::123:role/x", EnvRegion: "eu-west-1"}
	cfg, err := ParseConfig(json.RawMessage(`{}`), func(k string) string { return env[k] })
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RoleArn == "" || cfg.Region != "eu-west-1" {
		t.Errorf("env overrides not applied: %+v", cfg)
	}
}

func TestFactoryRegistered(t *testing.T) {
	// init() must have registered the AWS factory in the global catalog.
	f, ok := core.Factories()[core.ProviderAWS]
	if !ok {
		t.Fatal("AWS factory not registered by init()")
	}
	c, err := f(json.RawMessage(`{"role_arn":"arn:aws:iam::123:role/x","region":"eu-west-1"}`))
	if err != nil {
		t.Fatalf("factory build failed: %v", err)
	}
	if c.ID() != core.ProviderAWS {
		t.Errorf("unexpected connector id: %s", c.ID())
	}
}
