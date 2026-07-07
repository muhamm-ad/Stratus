package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"golang.org/x/oauth2"

	"github.com/muhamm-ad/stratus/core"
)

// Session holds the token set for one identity and refreshes it via x/oauth2.
type Session struct {
	client    *Client
	store     Store
	useDevice bool

	mu      sync.Mutex
	tok     *oauth2.Token
	idToken string
}

func NewSession(client *Client, store Store, useDevice bool) *Session {
	s := &Session{client: client, store: store, useDevice: useDevice}
	if t, id, err := store.Load(); err == nil && t != nil {
		s.tok, s.idToken = t, id
	}
	return s
}

// Login performs the browser flow (or device flow when configured), verifies
// the id_token, and persists the token set.
func (s *Session) Login(ctx context.Context, onCode func(core.DeviceCode)) error {
	var (
		tok *oauth2.Token
		err error
	)
	if s.useDevice {
		tok, err = s.client.DeviceLogin(ctx, onCode)
	} else {
		tok, err = s.client.BrowserLogin(ctx)
	}
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return core.ErrLoginCancelled
		}
		return err
	}

	id, _ := tok.Extra("id_token").(string)
	if id != "" && s.client.Verifier != nil {
		if _, verr := s.client.Verifier.Verify(ctx, id); verr != nil {
			return fmt.Errorf("id_token verification: %w", verr)
		}
	}

	s.mu.Lock()
	s.tok, s.idToken = tok, id
	s.mu.Unlock()
	_ = s.store.Save(tok, id)
	return nil
}

func (s *Session) IsAuthenticated() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tok.Valid()
}

// IDToken returns a valid id_token, refreshing the session if needed. x/oauth2's
// TokenSource performs the refresh_token grant transparently.
func (s *Session) IDToken(ctx context.Context) (string, error) {
	s.mu.Lock()
	tok, id := s.tok, s.idToken
	s.mu.Unlock()
	if tok == nil {
		return "", core.ErrNotAuthenticated
	}

	nt, err := s.client.OAuth.TokenSource(ctx, tok).Token()
	if err != nil {
		return "", core.ErrTokenExpired
	}
	if nt.AccessToken != tok.AccessToken { // refreshed
		if nid, ok := nt.Extra("id_token").(string); ok && nid != "" {
			id = nid
		}
		s.mu.Lock()
		s.tok, s.idToken = nt, id
		s.mu.Unlock()
		_ = s.store.Save(nt, id)
	}
	if id == "" {
		return "", core.ErrNotAuthenticated
	}
	return id, nil
}

// AccessToken silently acquires an access token for arbitrary scopes via the
// refresh token (e.g. the Azure ARM scope). Delegates the grant to x/oauth2.
func (s *Session) AccessToken(ctx context.Context, scopes ...string) (string, error) {
	s.mu.Lock()
	tok := s.tok
	s.mu.Unlock()
	if tok == nil || tok.RefreshToken == "" {
		return "", core.ErrNotAuthenticated
	}

	// x/oauth2 never sends a scope on refresh, so re-scoping the Entra refresh
	// token to (e.g.) the ARM resource must use our explicit RefreshGrant.
	t, err := RefreshGrant(ctx, http.DefaultClient, s.client.OAuth.Endpoint.TokenURL, s.client.OAuth.ClientID, tok.RefreshToken, scopes)
	if err != nil {
		return "", core.ErrTokenExpired
	}
	if t.RefreshToken != "" && t.RefreshToken != tok.RefreshToken {
		s.mu.Lock()
		if s.tok != nil {
			s.tok.RefreshToken = t.RefreshToken
			_ = s.store.Save(s.tok, s.idToken)
		}
		s.mu.Unlock()
	}
	return t.AccessToken, nil
}

func (s *Session) Logout(ctx context.Context) error {
	s.mu.Lock()
	s.tok, s.idToken = nil, ""
	s.mu.Unlock()
	return s.store.Clear()
}