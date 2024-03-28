package cache

import "github.com/pkg/errors"

var (
	// ErrNilCache for when a cache is nil.
	ErrNilCache = errors.New("cache cannot be nil")
	// ErrNilMetrics for when cache metrics are nil.
	ErrNilMetrics = errors.New("cache metrics cannot be nil")

	// ErrNilValueProvided for when we try to put a nil value in a cache.
	ErrNilValueProvided = errors.New("nil value provided on Put()")
	// ErrIncorrectType for when the state is of the incorrect type.
	ErrIncorrectType = errors.New("incorrect state type provided")
	// ErrNotFound for cache fetches that return a nil value.
	ErrNotFound = errors.New("not found in cache")
	// ErrNonExistingSyncCommitteeKey when sync committee key (root) does not exist in cache.
	ErrNonExistingSyncCommitteeKey = errors.New("does not exist sync committee key")

	// ErrNotFoundRegistration when validator registration does not exist in cache.
	ErrNotFoundRegistration = errors.Wrap(ErrNotFound, "no validator registered")

	// ErrAlreadyInProgress appears when attempting to mark a cache as in progress while it is
	// already in progress. The client should handle this error and wait for the in progress
	// data to resolve via Get.
	ErrAlreadyInProgress = errors.New("already in progress")

	// ErrCast for when a cast fails when the cache didn't get the expected type.
	ErrCast = errors.New("cast failed")
	// errNotSyncCommitteeIndexPosition for when the cache didn't get the expected type of syncCommitteeIndexPosition.
	errNotSyncCommitteeIndexPosition = errors.New("not of type SyncCommitteeIndexPosition")
	// errNotCommittees for when the cache didn't get the expected type of Committee.
	errNotCommittees = errors.New("not of type Committees")
)
