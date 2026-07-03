package auth

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/muhamm-ad/stratus/core"
)

// refreshWindow is how long before expiry a token is proactively refreshed.
const refreshWindow = 5 * time.Minute

// Session is a reusable OIDC session: it runs the Authorization Code + PKCE
// login, caches the token set, and refreshes it silently. Identity providers
// (Entra, Okta, …) embed a *Session and thus satisfy core.IdentityProvider —
// they differ only in endpoints and scope, supplied via OIDCConfig.
type Session struct {
	cfg   OIDCConfig
	store TokenStore
	open  func(string) error

	mu  sync.Mutex
	tok *TokenResponse
}

// NewSession builds a Session and loads any persisted token from the store.
func NewSession(cfg OIDCConfig, store TokenStore, open func(string) error) *Session {
	s := &Session{cfg: cfg, store: store, open: open}
	if t, err := store.Load(); err == nil && t != nil {
		s.tok = t
	}
	return s
}

// Login opens the browser once (PKCE + loopback) and caches the session.
func (s *Session) Login(ctx context.Context) error {
	tok, err := LoginAuthCode(ctx, s.cfg, s.open)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return core.ErrLoginCancelled
		}
		return err
	}
	s.mu.Lock()
	s.tok = tok
	s.mu.Unlock()
	_ = s.store.Save(tok)
	return nil
}

func (s *Session) IsAuthenticated() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tok.Valid(s.cfg.now())
}

// IDToken returns the current id_token, refreshing it if expiring soon.
func (s *Session) IDToken(ctx context.Context) (string, error) {
	s.mu.Lock()
	tok := s.tok
	s.mu.Unlock()
	if tok == nil || tok.IDToken == "" {
		return "", core.ErrNotAuthenticated
	}
	if tok.ExpiringSoon(s.cfg.now(), refreshWindow) {
		nt, err := RefreshToken(ctx, s.cfg, tok.RefreshToken)
		if err != nil {
			return "", core.ErrTokenExpired
		}
		if nt.RefreshToken == "" {
			nt.RefreshToken = tok.RefreshToken
		}
		s.mu.Lock()
		s.tok = nt
		s.mu.Unlock()
		_ = s.store.Save(nt)
		return nt.IDToken, nil
	}
	return tok.IDToken, nil
}

// AccessToken silently acquires an access token for an arbitrary scope via the
// refresh token (e.g. the ARM scope for Azure — only meaningful when the IdP can
// actually issue that scope, i.e. Entra).
func (s *Session) AccessToken(ctx context.Context, scope string) (string, error) {
	s.mu.Lock()
	var rt string
	if s.tok != nil {
		rt = s.tok.RefreshToken
	}
	s.mu.Unlock()
	if rt == "" {
		return "", core.ErrNotAuthenticated
	}
	cfg := s.cfg
	cfg.Scope = scope
	resp, err := RefreshToken(ctx, cfg, rt)
	if err != nil {
		return "", core.ErrTokenExpired
	}
	if resp.RefreshToken != "" {
		s.mu.Lock()
		if s.tok != nil {
			s.tok.RefreshToken = resp.RefreshToken
			_ = s.store.Save(s.tok)
		}
		s.mu.Unlock()
	}
	return resp.AccessToken, nil
}

func (s *Session) Logout(ctx context.Context) error {
	s.mu.Lock()
	s.tok = nil
	s.mu.Unlock()
	return s.store.Clear()
}