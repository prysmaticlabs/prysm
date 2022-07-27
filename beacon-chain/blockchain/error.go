package blockchain

import "github.com/pkg/errors"

var (
	// ErrInvalidPayload is returned when the payload is invalid
	ErrInvalidPayload = invalidBlock{error: errors.New("received an INVALID payload from execution engine")}
	// ErrInvalidBlockHashPayloadStatus is returned when the payload has invalid block hash.
	ErrInvalidBlockHashPayloadStatus = invalidBlock{error: errors.New("received an INVALID_BLOCK_HASH payload from execution engine")}
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
	errNotDescendantOfFinalized = invalidBlock{error: errors.New("not descendant of finalized checkpoint")}
)

// An invalid block is the block that fails state transition based on the core protocol rules.
// The beacon node shall not be accepting nor building blocks that branch off from an invalid block.
// Some examples of invalid blocks are:
// The block violates state transition rules.
// The block is deemed invalid according to execution layer client.
// The block violates certain fork choice rules (before finalized slot, not finalized ancestor)
type invalidBlock struct {
	invalidAncestorRoots [][32]byte
	error
	root [32]byte
}

type invalidBlockError interface {
	Error() string
	InvalidAncestorRoots() [][32]byte
	BlockRoot() [32]byte
}

// BlockRoot returns the invalid block root.
func (e invalidBlock) BlockRoot() [32]byte {
	return e.root
}

// InvalidAncestorRoots returns an optional list of invalid roots of the invalid block which leads up last valid root.
func (e invalidBlock) InvalidAncestorRoots() [][32]byte {
	return e.invalidAncestorRoots
}

// IsInvalidBlock returns true if the error has `invalidBlock`.
func IsInvalidBlock(e error) bool {
	if e == nil {
		return false
	}
	_, ok := e.(invalidBlockError)
	if !ok {
		return IsInvalidBlock(errors.Unwrap(e))
	}
	return true
}

// InvalidBlockRoot returns the invalid block root. If the error
// doesn't have an invalid blockroot. [32]byte{} is returned.
func InvalidBlockRoot(e error) [32]byte {
	if e == nil {
		return [32]byte{}
	}
	d, ok := e.(invalidBlockError)
	if !ok {
		return [32]byte{}
	}
	return d.BlockRoot()
}

// InvalidAncestorRoots returns a list of invalid roots up to last valid root.
func InvalidAncestorRoots(e error) [][32]byte {
	if e == nil {
		return [][32]byte{}
	}
	d, ok := e.(invalidBlockError)
	if !ok {
		return [][32]byte{}
	}
	return d.InvalidAncestorRoots()
}
