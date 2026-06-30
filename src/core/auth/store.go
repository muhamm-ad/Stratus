package auth

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/zalando/go-keyring"
)

// TokenResponse is the result of an OIDC token endpoint call.
type TokenResponse struct {
	AccessToken  string    `json:"access_token"`
	IDToken      string    `json:"id_token,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
	Scope        string    `json:"scope,omitempty"`
}

// Valid reports whether the access/id token is present and unexpired.
func (t *TokenResponse) Valid(now time.Time) bool {
	return t != nil && now.Before(t.ExpiresAt)
}

// ExpiringSoon reports whether the token expires within the window.
func (t *TokenResponse) ExpiringSoon(now time.Time, window time.Duration) bool {
	return t != nil && now.Add(window).After(t.ExpiresAt)
}

// TokenStore persists an OIDC session between runs.
type TokenStore interface {
	Save(*TokenResponse) error
	Load() (*TokenResponse, error)
	Clear() error
}

// KeyringStore persists tokens in the OS keychain via go-keyring.
type KeyringStore struct {
	service string
	user    string
}

// NewKeyringStore namespaces entries by a caller-supplied key.
func NewKeyringStore(key string) *KeyringStore {
	return &KeyringStore{service: "stratus", user: key}
}

func (s *KeyringStore) Save(t *TokenResponse) error {
	data, err := json.Marshal(t)
	if err != nil {
		return err
	}
	return keyring.Set(s.service, s.user, string(data))
}

func (s *KeyringStore) Load() (*TokenResponse, error) {
	raw, err := keyring.Get(s.service, s.user)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	var t TokenResponse
	if err := json.Unmarshal([]byte(raw), &t); err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *KeyringStore) Clear() error {
	err := keyring.Delete(s.service, s.user)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}

// MemoryStore is an in-process store for tests and keychain-less fallback.
type MemoryStore struct{ t *TokenResponse }

func (m *MemoryStore) Save(t *TokenResponse) error   { m.t = t; return nil }
func (m *MemoryStore) Load() (*TokenResponse, error) { return m.t, nil }
func (m *MemoryStore) Clear() error                  { m.t = nil; return nil }