package noop

import (
	"context"
)

type NoopCircuitBreaker[A any] struct {}

func (n NoopCircuitBreaker[A]) Protect(ctx context.Context, action func() (A, error)) (A, error) {
	return action()
}