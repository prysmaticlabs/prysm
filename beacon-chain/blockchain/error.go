package blockchain

import "github.com/pkg/errors"

var (
	// ErrInvalidPayload is returned when the payload is invalid
	ErrInvalidPayload = errors.New("received an INVALID payload from execution engine")
	// ErrInvalidBlockHashPayloadStatus is returned when the payload has invalid block hash.
	ErrInvalidBlockHashPayloadStatus = errors.New("received an INVALID_BLOCK_HASH payload from execution engine")
	// ErrUndefinedExecutionEngineError is returned when the execution engine returns an error that is not defined
	ErrUndefinedExecutionEngineError = errors.New("received an undefined ee error")
	// errNilFinalizedInStore is returned when a nil finalized checkpt is returned from store.
	errNilFinalizedInStore = errors.New("nil finalized checkpoint returned from store")
	// errNilFinalizedCheckpoint is returned when a nil finalized checkpt is returned from a state.
	errNilFinalizedCheckpoint = errors.New("nil finalized checkpoint returned from state")
	// errNilJustifiedCheckpoint is returned when a nil justified checkpt is returned from a state.
	errNilJustifiedCheckpoint = errors.New("nil finalized checkpoint returned from state")
	// errInvalidNilSummary is returned when a nil summary is returned from the DB.
	errInvalidNilSummary = errors.New("nil summary returned from the DB")
	// errWrongBlockCount is returned when the wrong number of blocks or block roots is used
	errWrongBlockCount = errors.New("wrong number of blocks or block roots")
	// block is not a valid optimistic candidate block
	errNotOptimisticCandidate = errors.New("block is not suitable for optimistic sync")
	// errBlockNotFoundInCacheOrDB is returned when a block is not found in the cache or DB.
	errBlockNotFoundInCacheOrDB = errors.New("block not found in cache or db")
	// errNilStateFromStategen is returned when a nil state is returned from the state generator.
	errNilStateFromStategen = errors.New("justified state can't be nil")
	// errWSBlockNotFound is returned when a block is not found in the WS cache or DB.
	errWSBlockNotFound = errors.New("weak subjectivity root not found in db")
	// errWSBlockNotFoundInEpoch is returned when a block is not found in the WS cache or DB within epoch.
	errWSBlockNotFoundInEpoch = errors.New("weak subjectivity root not found in db within epoch")
	// errNotDescendantOfFinalized is returned when a block is not a descendant of the finalized checkpoint
	errNotDescendantOfFinalized = invalidBlock{errors.New("not descendant of finalized checkpoint")}
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
