package cache

import "errors"

var (
	// ErrNilValueProvided for when we try to put a nil value in a cache.
	ErrNilValueProvided = errors.New("nil value provided on Put()")
	// ErrIncorrectType for when the state is of the incorrect type.
	ErrIncorrectType = errors.New("incorrect state type provided")
	// ErrNotFound for cache fetches that return a nil value.
	ErrNotFound = errors.New("not found in cache")
	// ErrNonExistingSyncCommitteeKey when sync committee key (root) does not exist in cache.
	ErrNonExistingSyncCommitteeKey   = errors.New("does not exist sync committee key")
	errNotSyncCommitteeIndexPosition = errors.New("not syncCommitteeIndexPosition struct")
)
