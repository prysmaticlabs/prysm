package state

import "errors"

var (
	ErrNilParticipation = errors.New("nil epoch participation in state")
	// ErrNilValidatorsInState returns when accessing validators in the state while the state has a
	// nil slice for the validators field.
	ErrNilValidatorsInState = errors.New("state has nil validator slice")
)
