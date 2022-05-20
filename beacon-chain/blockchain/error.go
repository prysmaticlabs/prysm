package blockchain

import "github.com/pkg/errors"

var (
	// errNilFinalizedInStore is returned when a nil finalized checkpt is returned from store.
	errNilFinalizedInStore = errors.New("nil finalized checkpoint returned from store")
	// errInvalidNilSummary is returned when a nil summary is returned from the DB.
	errInvalidNilSummary = errors.New("nil summary returned from the DB")
	// errWrongBlockCount is returned when the wrong number of blocks or
	// block roots is used
	errWrongBlockCount = errors.New("wrong number of blocks or block roots")
	// block is not a valid optimistic candidate block
	errNotOptimisticCandidate = errors.New("block is not suitable for optimistic sync")
)
