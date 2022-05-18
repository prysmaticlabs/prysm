package blockchain

import "github.com/pkg/errors"

var (
	// errNilJustifiedInStore is returned when a nil justified checkpt is returned from store.
	errNilJustifiedInStore = errors.New("nil justified checkpoint returned from store")
	// errNilBestJustifiedInStore is returned when a nil justified checkpt is returned from store.
	errNilBestJustifiedInStore = errors.New("nil best justified checkpoint returned from store")
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

// An invalid block is the block that fails state transition based on the core protocol rules.
// The beacon node shall not be accepting and builder blocks that branch off of the invalid block.
// Some examples of invalid blocks are:
// The block violates state transition rules.
// The block is deemed invalid according to execution layer client.
// The block violates certain fork choice rules (before finalized slot, not finalized ancestor)
type invalidBlock struct {
	error
}

type invalidBlockError interface {
	Error() string
	InvalidBlock() bool
}

// InvalidBlock returns true for `invalidBlock`.
func (e invalidBlock) InvalidBlock() bool {
	return true
}

// IsInvalidBlock returns true if the error has `invalidBlock`.
func IsInvalidBlock(e error) bool {
	d, ok := e.(invalidBlockError)
	if !ok {
		uw := errors.Unwrap(e)
		if uw != nil {
			return IsInvalidBlock(uw)
		}
	}
	return ok && d.InvalidBlock()
}
