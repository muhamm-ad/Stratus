package aws

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/zalando/go-keyring"
)

// Token is a stored SSO session. It includes the client registration so the
// session can be refreshed without re-registering.
type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
	Region       string    `json:"region"`
	StartURL     string    `json:"start_url"`
	ClientID     string    `json:"client_id"`
	ClientSecret string    `json:"client_secret"`
}

// Valid reports whether the token is present and not expired at time now.
func (t *Token) Valid(now time.Time) bool {
	return t != nil && t.AccessToken != "" && now.Before(t.ExpiresAt)
}

// expiringSoon reports whether the token expires within the given window.
func (t *Token) expiringSoon(now time.Time, window time.Duration) bool {
	return t != nil && now.Add(window).After(t.ExpiresAt)
}

// TokenStore persists the SSO session between runs.
type TokenStore interface {
	Save(*Token) error
	Load() (*Token, error)
	Clear() error
}

// keyringStore persists tokens in the OS keychain (macOS Keychain, Windows
// Credential Manager, libsecret on Linux) via go-keyring. This is deliberately
// more secure than the AWS CLI, which writes tokens as plaintext JSON under
// ~/.aws/sso/cache.
type keyringStore struct {
	service string
	user    string
}

// newKeyringStore namespaces entries per SSO start URL so multiple Stratus
// configurations do not collide.
func newKeyringStore(startURL string) *keyringStore {
	sum := sha1.Sum([]byte(startURL))
	return &keyringStore{service: "stratus-aws", user: hex.EncodeToString(sum[:])}
}

func (s *keyringStore) Save(t *Token) error {
	data, err := json.Marshal(t)
	if err != nil {
		return err
	}
	return keyring.Set(s.service, s.user, string(data))
}

func (s *keyringStore) Load() (*Token, error) {
	raw, err := keyring.Get(s.service, s.user)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	var t Token
	if err := json.Unmarshal([]byte(raw), &t); err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *keyringStore) Clear() error {
	err := keyring.Delete(s.service, s.user)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}

// memoryStore is an in-process store used in tests and as a fallback when no
// OS keychain is available.
type memoryStore struct{ t *Token }

func (m *memoryStore) Save(t *Token) error   { m.t = t; return nil }
func (m *memoryStore) Load() (*Token, error) { return m.t, nil }
func (m *memoryStore) Clear() error          { m.t = nil; return nil }
