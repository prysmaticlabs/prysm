package cache

import "errors"

var (
	// ErrNotFound for cache fetches that return a nil value.
	ErrNotFound = errors.New("not found in cache")
	// ErrNonExistingSyncCommitteeKey when sync committee key (root) does not exist in cache.
	ErrNonExistingSyncCommitteeKey = errors.New("does not exist sync committee key")
	// ErrNonExistingBalanceKey when balance key does not exist in cache.
	ErrNonExistingBalanceKey         = errors.New("does not exist balance key")
	errNotSyncCommitteeIndexPosition = errors.New("not syncCommitteeIndexPosition struct")
)
