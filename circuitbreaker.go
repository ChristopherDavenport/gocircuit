// GoCircuit implements the primary inferfaces and types for
// the circuit breaker pattern.

package gocircuit

import (
	"context"
)

// CircuitBreaker is an interface that represents a circuit breaker.
//
// Its only function is to show that
type CircuitBreaker[A any] interface {
	Protect(ctx context.Context, action func() (A, error)) (A, error)

  // TODO: Two-Stage Approach
	// Check(ctx context.Context) (bool, error)
	// Report(ctx context.Context, success bool) error
}