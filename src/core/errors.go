package core

import "errors"

// Typed errors let the UI react precisely instead of string-matching.
var (
	// ErrNotImplemented marks a capability not yet built for a provider.
	ErrNotImplemented = errors.New("stratus: not implemented")

	// ErrLoginCancelled is returned when the user aborts the Entra login.
	ErrLoginCancelled = errors.New("stratus: login cancelled")

	// ErrNotAuthenticated is returned when an operation needs a session but
	// none exists.
	ErrNotAuthenticated = errors.New("stratus: not authenticated")

	// ErrTokenExpired indicates a token expired and could not be refreshed.
	ErrTokenExpired = errors.New("stratus: token expired")

	// ErrExchange indicates a provider credential exchange failed
	// (AssumeRoleWithWebIdentity, STS token exchange, ARM acquisition).
	ErrExchange = errors.New("stratus: credential exchange failed")
)