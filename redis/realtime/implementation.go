package realtime


import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/christopherdavenport/gocircuit"
)



type StateStruct struct {
	State gocircuit.State
	TimeOpen time.Time
}


var (
	closedZero = StateStruct{State: gocircuit.StateClosed, TimeOpen: time.Time{}}
)

type CircuitBreakerError string

func (e CircuitBreakerError) Error() string {
	return string(e)
}

const CircuitBreakerOpen CircuitBreakerError = "circuit breaker is open/half-open"

// type CircuitBreaker interface {
// 	Execute(req func() (interface{}, error)) (interface{}, error)
// }

func StateStructToString(s StateStruct) (string, error) {
	stateString, err := gocircuit.StateToString(s.State)
	if err != nil {
		return "", err
	}

  switch s.State {
		case gocircuit.StateClosed:
			stateString = fmt.Sprintf("%s %d", stateString, s.TimeOpen.UnixNano())
		case gocircuit.StateHalfOpen:
			stateString = fmt.Sprintf("%s %d", stateString, s.TimeOpen.UnixNano())
		case gocircuit.StateOpen:
			stateString = fmt.Sprintf("%s %d", stateString, s.TimeOpen.UnixNano())
		default:
			return "", fmt.Errorf("unknown state: %d", s.State)
	}
	return stateString, nil
}

func StateStructFromString(s string) (StateStruct, error) {
	parts := strings.Split(s, " ")
	if len(parts) > 2 {
		return StateStruct{}, fmt.Errorf("invalid state string: %s", s)
	}
	state, err := gocircuit.StateFromString(parts[0])
	if err != nil {
		return StateStruct{}, err
	}
	var timeOpen time.Time
	if (len(parts) == 2){
		timeOpen, err = timeFromString(parts[1])
		if err != nil {
		return StateStruct{}, err
		}
	} else {
		timeOpen = time.Time{}
	}
	return StateStruct{State: state, TimeOpen: timeOpen}, nil
}






func stateKey(prefix string, key string) string {
	return fmt.Sprintf("%s:state:%s", prefix, key)
}

func requestKey(prefix string, key string) string {
	return fmt.Sprintf("%s:req:%s", prefix, key)
}
func failureKey(prefix string, key string) string {
	return fmt.Sprintf("%s:fail:%s", prefix, key)
}
func consecutiveFailureKey(prefix string, key string) string {
	return fmt.Sprintf("%s:confail:%s", prefix, key)
}

func successKey(prefix string, key string) string {
	return fmt.Sprintf("%s:succ:%s", prefix, key)
}

func consecutiveSuccessKey(prefix string, key string) string {
	return fmt.Sprintf("%s:consucc:%s", prefix, key)
}

func halfOpenKey(prefix string, key string) string {
	return fmt.Sprintf("%s:half:%s", prefix, key)
}

func timeFromString(s string) (time.Time, error) {
	if (s == "") {
		return time.Time{}, fmt.Errorf("empty string")
	}
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time string: %s", s)
	}
	return time.Unix(0, i), nil
}

type circuitBreakerInfo struct {
	Id uuid.UUID
	State gocircuit.State
  TimeOpen time.Time

	TotalRequests int64
	TotalSuccesses int64
	TotalFailures int64
	ConsecutiveSuccesses int64
	ConsecutiveFailures int64
}

func (cbi circuitBreakerInfo) Counts()	Counts {
	return Counts{
		Requests: cbi.TotalRequests,
		TotalSuccesses: cbi.TotalSuccesses,
		TotalFailures: cbi.TotalFailures,
		ConsecutiveSuccesses: cbi.ConsecutiveSuccesses,
		ConsecutiveFailures: cbi.ConsecutiveFailures,
	}
}


func (cbi circuitBreakerInfo) String() string {
	return fmt.Sprintf("CircuitBreakerInfo(Id: %s, State: %s, TimeOpen: %s, TotalRequests: %d, TotalSuccesses: %d, TotalFailures: %d, ConsecutiveSuccesses: %d, ConsecutiveFailures: %d)", cbi.Id, cbi.State, cbi.TimeOpen, cbi.TotalRequests, cbi.TotalSuccesses, cbi.TotalFailures, cbi.ConsecutiveSuccesses, cbi.ConsecutiveFailures)
}

func getInformation(client *redis.Client, ctx context.Context, key string, systime time.Time, settings CircuitBreakerSettings) (*circuitBreakerInfo, error) {
	pipe := client.Pipeline()

	id := uuid.New()

	clearTimings(pipe, ctx, key, systime, settings)
	stateCmd := pipe.Get(ctx, stateKey(settings.Prefix, key))
	requestCountCmd := pipe.ZCard(ctx, requestKey(settings.Prefix, key))
	successCountCmd := pipe.ZCard(ctx, successKey(settings.Prefix, key))
	failureCountCmd := pipe.ZCard(ctx, failureKey(settings.Prefix, key))
	consecutiveSuccessCountCmd := pipe.ZCard(ctx, consecutiveSuccessKey(settings.Prefix, key))
	consecutiveFailureCountCmd := pipe.ZCard(ctx, consecutiveFailureKey(settings.Prefix, key))

	_, err := pipe.Exec(ctx)

	if err != nil && err != redis.Nil {
		return nil, err
	}
	var stateStruct StateStruct
	stateString, err := stateCmd.Result()

	if err != nil && err != redis.Nil {
		return nil, err
	}


	if err == redis.Nil {
		stateStruct = closedZero // If they key is unset, it is closed.
	} else {
		stateStruct, err = StateStructFromString(stateString)
		if err != nil {
			return nil, err
		}
	}



	requestCount, err := requestCountCmd.Result()
	if err != nil {
		return nil, err
	}

	successCount, err := successCountCmd.Result()
	if err != nil {
		return nil, err
	}

	failureCount, err := failureCountCmd.Result()
	if err != nil {
		return nil, err
	}

	consecutiveSuccessCount, err := consecutiveSuccessCountCmd.Result()
	if err != nil {
		return nil, err
	}

	consecutiveFailureCount, err := consecutiveFailureCountCmd.Result()
	if err != nil {
		return nil, err
	}

	return &circuitBreakerInfo{
		Id: id,
		State: stateStruct.State,
		TimeOpen: stateStruct.TimeOpen,

		TotalRequests: requestCount,
		TotalSuccesses: successCount,
		TotalFailures: failureCount,
		ConsecutiveSuccesses: consecutiveSuccessCount,
		ConsecutiveFailures: consecutiveFailureCount,

	}, nil

}

func clear(client *redis.Client, ctx context.Context, key string, settings CircuitBreakerSettings) error {
	pipe := client.Pipeline()
	pipe.Del(ctx, stateKey(settings.Prefix, key))
	pipe.Del(ctx, requestKey(settings.Prefix, key))
	pipe.Del(ctx, failureKey(settings.Prefix, key))
	pipe.Del(ctx, successKey(settings.Prefix, key))
	pipe.Del(ctx, consecutiveSuccessKey(settings.Prefix, key))
	pipe.Del(ctx, consecutiveFailureKey(settings.Prefix, key))
	_, err := pipe.Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}

func clearTimings(pipe redis.Pipeliner, ctx context.Context, key string, systime time.Time, settings CircuitBreakerSettings) {
	currentWindow := systime.UnixNano() - settings.Interval.Nanoseconds()
	pipe.ZRemRangeByScore(ctx, requestKey(settings.Prefix, key), "-inf", strconv.FormatInt(currentWindow, 10))
	pipe.ZRemRangeByScore(ctx, failureKey(settings.Prefix, key), "-inf", strconv.FormatInt(currentWindow, 10))
	pipe.ZRemRangeByScore(ctx, successKey(settings.Prefix, key), "-inf", strconv.FormatInt(currentWindow, 10))
	pipe.ZRemRangeByScore(ctx, consecutiveSuccessKey(settings.Prefix, key), "-inf", strconv.FormatInt(currentWindow, 10))
	pipe.ZRemRangeByScore(ctx, consecutiveFailureKey(settings.Prefix, key), "-inf", strconv.FormatInt(currentWindow, 10))

}


func registerSuccess(client *redis.Client, ctx context.Context, key string, settings CircuitBreakerSettings) error {
	pipe := client.Pipeline()
	stateStruct := StateStruct{
		State: gocircuit.StateClosed,
		TimeOpen: time.Time{},
	}
	stateStructString, err := StateStructToString(stateStruct)
	if err != nil {
		return err
	}


	systime := time.Now()
	pipe.Set(ctx, stateKey(settings.Prefix, key), stateStructString, settings.RedisKeyTimeout)
	pipe.ZAdd(ctx, requestKey(settings.Prefix, key), redis.Z{
		Member: systime,
		Score: float64(systime.UnixNano()),
	})
	pipe.ZAdd(ctx, successKey(settings.Prefix, key), redis.Z{
		Member: systime,
		Score: float64(systime.UnixNano()),
	})
	pipe.ZAdd(ctx, consecutiveSuccessKey(settings.Prefix, key), redis.Z{
		Member: systime,
		Score: float64(systime.UnixNano()),
	})
	pipe.Del(ctx, consecutiveFailureKey(settings.Prefix, key))

	pipe.PExpire(ctx, requestKey(settings.Prefix, key), settings.RedisKeyTimeout)
	pipe.PExpire(ctx, failureKey(settings.Prefix, key), settings.RedisKeyTimeout)
	pipe.PExpire(ctx, successKey(settings.Prefix, key), settings.RedisKeyTimeout)
	pipe.PExpire(ctx, consecutiveSuccessKey(settings.Prefix, key), settings.RedisKeyTimeout)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}



// How we add a failure.
func registerFailure(client *redis.Client, ctx context.Context, key string, settings CircuitBreakerSettings) error {
	pipe := client.Pipeline()


	systime := time.Now()
	pipe.ZAdd(ctx, requestKey(settings.Prefix, key), redis.Z{
		Member: systime,
		Score: float64(systime.UnixNano()),
	})
	pipe.ZAdd(ctx, failureKey(settings.Prefix, key), redis.Z{
		Member: systime,
		Score: float64(systime.UnixNano()),
	})
	pipe.ZAdd(ctx, consecutiveFailureKey(settings.Prefix, key), redis.Z{
		Member: systime,
		Score: float64(systime.UnixNano()),
	})
	pipe.Del(ctx, consecutiveSuccessKey(settings.Prefix, key))

	pipe.PExpire(ctx, stateKey(settings.Prefix, key), settings.RedisKeyTimeout)
	pipe.PExpire(ctx, requestKey(settings.Prefix, key), settings.RedisKeyTimeout)
	pipe.PExpire(ctx, failureKey(settings.Prefix, key), settings.RedisKeyTimeout)
	pipe.PExpire(ctx, successKey(settings.Prefix, key), settings.RedisKeyTimeout)
	pipe.PExpire(ctx, consecutiveSuccessKey(settings.Prefix, key), settings.RedisKeyTimeout)
	pipe.PExpire(ctx, consecutiveFailureKey(settings.Prefix, key), settings.RedisKeyTimeout)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}


func empty[A any]() A {
	var a A
	return a
}



func runClosed[A any](client *redis.Client, ctx context.Context, key string, settings CircuitBreakerSettings, action func() (A, error)) (A, error) {
	out, initialErr := action()
	if initialErr != nil && !settings.IsSuccessful(initialErr) {
		// TODO - Add logging
		_ = registerFailure(client, ctx, key, settings)
	} else {
		_ = registerSuccess(client, ctx, key, settings)
	}
	return out, initialErr
}

func runOpenAndHalf[A any]() (A, error, bool) {
	return empty[A](), CircuitBreakerOpen, false
}
func setToOpen(client *redis.Client, ctx context.Context, key string, settings CircuitBreakerSettings, oldState StateStruct) (error, bool) {
	systime := time.Now() // TODO - Use last operation time, rather than now

	state := StateStruct{
		State: gocircuit.StateOpen,
		TimeOpen: systime.Add(settings.OpenTimeout),
	}
	stateString, err := StateStructToString(state)
	if err != nil {
		return err, false
	}

	txf := func(tx *redis.Tx) error {
		currentStateString, err := tx.Get(ctx, stateKey(settings.Prefix, key)).Result()
		if err != nil && err != redis.Nil {
			return err
		}
		var currentState StateStruct
		if err != redis.Nil {
			currentState, err = StateStructFromString(currentStateString)
			if err != nil {
				return err
			}
		}
		// fmt.Println("currentState:", currentState, ", oldState:", oldState, "state:", state)
		if currentState.State != oldState.State && currentState.TimeOpen != oldState.TimeOpen && err != redis.Nil {

			return redis.TxFailedErr // This triggers same behavior as a failed transaction.
		}

		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Set(ctx, stateKey(settings.Prefix, key), stateString, settings.RedisKeyTimeout)
			return nil
		})
		if err == nil {
			settings.OnStateChange(currentState.State, state.State)
		}


		return err
	}

	err = client.Watch(ctx, txf, stateKey(settings.Prefix, key))
	if err != nil {
		if err == redis.TxFailedErr {
			return err, true
		}
		return err, false
	}
	return CircuitBreakerOpen, false
}

func setToHalfOpen[A any](client *redis.Client, ctx context.Context, key string, settings CircuitBreakerSettings, oldState StateStruct, f func() (A, error)) (A, error, bool) {
	systime := time.Now() // TODO - Use last operation time, rather than now

	state := StateStruct{
		State: gocircuit.StateHalfOpen,
		TimeOpen: systime.Add(settings.OpenTimeout),
	}
	stateString, err := StateStructToString(state)
	if err != nil {
		return empty[A](), err, false
	}

	txf := func(tx *redis.Tx) error {
		currentStateString, err := tx.Get(ctx, stateKey(settings.Prefix, key)).Result()
		if err != nil && err != redis.Nil {
			return err
		}


		var currentState StateStruct
		if err != redis.Nil {
			currentState, err = StateStructFromString(currentStateString)
			if err != nil {
				return err
			}
		}
		// fmt.Println("currentState:", currentState, ", oldState:", oldState, "state:", state)
		if currentState.State != oldState.State && currentState.TimeOpen != oldState.TimeOpen && err != redis.Nil {

			return redis.TxFailedErr // This triggers same behavior as a failed transaction.
		}

		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Set(ctx, stateKey(settings.Prefix, key), stateString, settings.RedisKeyTimeout)
			pipe.ZAdd(ctx, halfOpenKey(settings.Prefix, key), redis.Z{
				Member: systime,
				Score: float64(systime.UnixNano()),
			})
			return nil
		})
		if err == nil {
			settings.OnStateChange(currentState.State, state.State)
		}
		return err
	}

	err = client.Watch(ctx, txf, stateKey(settings.Prefix, key), halfOpenKey(settings.Prefix, key))
	if err != nil {
		return empty[A](), err, false
	}
	value, initErr := f()
	if initErr != nil && !settings.IsSuccessful(initErr) {
		err, retry := setToOpen(client, ctx, key, settings, state)

		if err != redis.TxFailedErr && err != nil {
			return value, err, false
		}
		err = client.ZRem(ctx, halfOpenKey(settings.Prefix, key), systime).Err()
		if err != nil {
			return value, err, retry
		}
		return value, err, retry
	}
	_ = clear(client, ctx, key, settings)
	settings.OnStateChange(state.State, gocircuit.StateClosed)

	return value, initErr, false
}



func protectInternal[A any](client *redis.Client, ctx context.Context, key string, settings CircuitBreakerSettings, action func() (A, error)) (A, error, bool) {
	now := time.Now()
	cbi, err := getInformation(client, ctx, key, now, settings)
	if err != nil {
		return empty[A](), err, false
	}
	// fmt.Println(cbi)

	if cbi.State == gocircuit.StateClosed {
		if !settings.ReadyToTrip(cbi.Counts()) { // If Closed and not ready to trip then run the action.
			// fmt.Println("Running Closed")
			a, err := runClosed(client, ctx, key, settings, action)
			return a, err, false
		}
		// Ready to Trip
		// fmt.Println("Ready to Trip")
		err, retry := setToOpen(client, ctx, key, settings, StateStruct{State: cbi.State, TimeOpen: cbi.TimeOpen})
		if err != nil {
			// fmt.Println("Error from setToOpen:", err)
			return empty[A](), err, retry
		}
		return empty[A](), CircuitBreakerOpen, false
	} else if cbi.State == gocircuit.StateOpen {
		systime := time.Now()
		diff := cbi.TimeOpen.Sub(systime)
		if diff <= 0 { // If Open and time open has exceeded the OpenTimeout then attempt to change to Half Open
			// fmt.Println("Changing to Half Open")
			return setToHalfOpen[A](client, ctx, key, settings, StateStruct{State: cbi.State, TimeOpen: cbi.TimeOpen}, action)

			// Change to Half Open
		}
	}

	return runOpenAndHalf[A]()

}

func protect[A any](client *redis.Client, ctx context.Context, key string, settings CircuitBreakerSettings, action func() (A, error)) (A, error) {
	var value A
	var err error
	loop := true
	for loop {
		value, err, loop = protectInternal[A](client, ctx, key, settings, action)
	}
	return value, err
}