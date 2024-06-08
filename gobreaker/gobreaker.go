package gobreaker

import (
	"context"
	"github.com/christopherdavenport/gocircuit"
	gb "github.com/sony/gobreaker/v2"
)

type goBreakerCircuit[A any] struct {
	circuit *gb.CircuitBreaker[A]
}

func (g *goBreakerCircuit[A]) Protect(ctx context.Context, action func() (A, error)) (A, error) {
	return g.circuit.Execute(action)
}

func NewGoBreakerCircuitBreaker[A any](breaker *gb.CircuitBreaker[A]) gocircuit.CircuitBreaker[A] {
	return &goBreakerCircuit[A]{circuit: breaker}
}