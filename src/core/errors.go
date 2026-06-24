package core

import "errors"

// Typed errors let the UI react precisely instead of string-matching.
var (
	// ErrNotImplemented marks a capability not yet built for a provider.
	ErrNotImplemented = errors.New("stratus: not implemented")

	// ErrLoginCancelled is returned when the user aborts the auth flow.
	// The UI should return to idle without showing an error toast.
	ErrLoginCancelled = errors.New("stratus: login cancelled")

	// ErrNotAuthenticated is returned when an operation needs a session but
	// no valid one exists.
	ErrNotAuthenticated = errors.New("stratus: not authenticated")

	// ErrTokenExpired indicates the session token has expired and could not
	// be refreshed silently; the user must log in again.
	ErrTokenExpired = errors.New("stratus: session token expired")

	// ErrNoEntitlements is returned when authentication succeeds but the user
	// has no accessible accounts.
	ErrNoEntitlements = errors.New("stratus: no accessible accounts")
)
