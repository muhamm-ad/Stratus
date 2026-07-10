package auth

import (
	"encoding/json"
	"errors"

	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

// persisted serializes the oauth2.Token together with the id_token, which
// oauth2.Token keeps only in its (non-serialized) Extra fields.
type persisted struct {
	Token   *oauth2.Token `json:"token"`
	IDToken string        `json:"id_token,omitempty"`
}

// Store persists a token set in the OS keychain (via go-keyring).
type Store struct {
	Service string // keychain service, e.g. "stratus"
	Key     string // per-identity key, e.g. "oidc:entra"
}

func (s Store) Save(t *oauth2.Token, idToken string) error {
	b, err := json.Marshal(persisted{Token: t, IDToken: idToken})
	if err != nil {
		return err
	}
	return keyring.Set(s.Service, s.Key, string(b))
}

func (s Store) Load() (*oauth2.Token, string, error) {
	raw, err := keyring.Get(s.Service, s.Key)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	var p persisted
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return nil, "", err
	}
	return p.Token, p.IDToken, nil
}

func (s Store) Clear() error {
	err := keyring.Delete(s.Service, s.Key)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}