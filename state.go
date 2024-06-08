package gocircuit

import (
	"fmt"
)

// State is a type that represents a state of CircuitBreaker.
type State int

// These constants are states of CircuitBreaker.
const (
	StateClosed State = iota
	StateHalfOpen
	StateOpen
)

func StateFromString(s string) (State, error) {
	switch s {
	case "closed":
		return StateClosed, nil
	case "half":
		return StateHalfOpen, nil
	case "open":
		return StateOpen, nil
	default:
		return StateClosed, fmt.Errorf("unknown state: %s", s)
	}
}

func StateToString(s State) (string, error) {
	switch s {
	case StateClosed:
		return "closed", nil
	case StateHalfOpen:
		return "half", nil
	case StateOpen:
		return "open", nil
	default:
		return "closed", fmt.Errorf("unknown state: %d", s)
	}
}

func (s State) String() string {
	stateString, err := StateToString(s)
	if err != nil {
		return "unknown state"
	}
	return stateString
}