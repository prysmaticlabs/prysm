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

// An invalid block is the block that fails state transition based on the core protocol rules.
// The beacon node shall not be accepting nor building blocks that branch off from an invalid block.
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
	if e == nil {
		return false
	}
	d, ok := e.(invalidBlockError)
	if !ok {
		return IsInvalidBlock(errors.Unwrap(e))
	}
	return d.InvalidBlock()
}
