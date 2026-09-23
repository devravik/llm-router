package llmrouter

import "errors"

// Sentinel errors returned by Router methods. Callers can check for a
// specific condition with errors.Is.
var (
	// ErrNoRoutes is returned by Next when no route is eligible for
	// selection.
	ErrNoRoutes = errors.New("llmrouter: no eligible routes")

	// ErrNotFound is returned by Remove, Enable, and Disable when the
	// given route ID is not registered.
	ErrNotFound = errors.New("llmrouter: route not found")

	// ErrDuplicateRoute is returned by Add when the route ID is already
	// registered.
	ErrDuplicateRoute = errors.New("llmrouter: route already registered")

	// ErrInvalidRoute is returned by Add when the route has an empty ID
	// or a negative Weight.
	ErrInvalidRoute = errors.New("llmrouter: invalid route")
)
