package realtime


import (
	"context"
	"time"
	"github.com/redis/go-redis/v9"
  "github.com/christopherdavenport/gocircuit"
)

type realtimeRedisCircuitBreakerSimple[A any] struct {
	client *redis.Client
	settings CircuitBreakerSettings
	key string
}

func (cb realtimeRedisCircuitBreakerSimple[A]) Protect(ctx context.Context, action func() (A, error)) (A, error) {
	return protect(cb.client, ctx, cb.key, cb.settings, action)
}


func NewRealtimeRedisCircuitBreaker[A any](client *redis.Client, key string, settings CircuitBreakerSettings) gocircuit.CircuitBreaker[A] {
	return &realtimeRedisCircuitBreakerSimple[A]{
		client: client,
		settings: settings,
		key: key,
	}
}


type CircuitBreakerSettings struct {
	Prefix string
	RedisKeyTimeout time.Duration // The period of time after which the state of the circuit breaker is considered stale and is reset to closed.

	Interval time.Duration // The period of time over which requests are counted.
	OpenTimeout time.Duration // The period of time after which the circuit breaker transitions from open to half-open.


	ReadyToTrip	func(info Counts) bool
	OnStateChange func(old gocircuit.State, new gocircuit.State) error
	IsSuccessful  func(err error) bool

}

type Counts struct {
	Requests int64
	TotalSuccesses int64
	TotalFailures int64
	ConsecutiveSuccesses int64
	ConsecutiveFailures int64
}

