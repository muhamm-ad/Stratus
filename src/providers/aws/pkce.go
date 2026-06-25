package aws

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// pkcePair holds a PKCE verifier and its derived S256 challenge (RFC 7636).
type pkcePair struct {
	Verifier  string
	Challenge string
}

// newPKCE generates a high-entropy code verifier and its S256 challenge.
// The verifier is a 43-character base64url string (32 random bytes), which is
// within the RFC 7636 length bounds of 43..128.
func newPKCE() (pkcePair, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return pkcePair{}, fmt.Errorf("aws: pkce entropy: %w", err)
	}
	verifier := base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	return pkcePair{Verifier: verifier, Challenge: challenge}, nil
}

// newState returns a random opaque value used to defend the loopback callback
// against CSRF (RFC 6749 §10.12).
func newState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("aws: state entropy: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
