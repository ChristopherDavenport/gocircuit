package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/christopherdavenport/gocircuit"
	gcredis "github.com/christopherdavenport/gocircuit/redis/realtime"
	"github.com/redis/go-redis/v9"
	"time"
)

func main() {
	key := "test"
	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	circuitBreakerSettings := gcredis.CircuitBreakerSettings{
		Prefix:          "circuitBreaker",
		RedisKeyTimeout: 5 * time.Minute,

		Interval:    1 * time.Minute,
		OpenTimeout: 10 * time.Second,

		OnStateChange: func(old gocircuit.State, new gocircuit.State) error {
			fmt.Printf("State changed from %s to %s\n", old, new)
			return nil
		},
		ReadyToTrip: func(info gcredis.Counts) bool {
			return info.TotalFailures > 1
		},
		IsSuccessful: func(err error) bool {
			return err == nil
		},
	}

	err := rdb.FlushAll(ctx).Err()
	if err != nil {
		fmt.Println(err)
		return
	}

	cb := gcredis.NewRealtimeRedisCircuitBreaker[interface{}](rdb, key, circuitBreakerSettings)

	_, err = cb.Protect(ctx, func() (interface{}, error) {
		return nil, errors.New("error 1")
	})
	if err != nil {
		fmt.Println(err)
	}
	_, err = cb.Protect(ctx, func() (interface{}, error) {
		return nil, errors.New("error 2")
	})
	if err != nil {
		fmt.Println(err)
	}
	_, err = cb.Protect(ctx, func() (interface{}, error) {
		return nil, errors.New("error 3")
	})
	if err != nil {
		fmt.Println(err)
	}

	time.Sleep(11 * time.Second)
	fmt.Println("Time to sleep, waiting for half open")

	_, err = cb.Protect(ctx, func() (interface{}, error) {
		return nil, nil
	})
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("Success 4")

	}

	_, err = cb.Protect(ctx, func() (interface{}, error) {
		return nil, nil
	})
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("Success 5")
	}

	_, err = cb.Protect(ctx, func() (interface{}, error) {
		return nil, errors.New("error 6")
	})
	if err != nil {
		fmt.Println(err)
	}

	_, err = cb.Protect(ctx, func() (interface{}, error) {
		return nil, errors.New("error 7")
	})
	if err != nil {
		fmt.Println(err)
	}
}
