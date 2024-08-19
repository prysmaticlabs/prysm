package verification

import "github.com/prysmaticlabs/prysm/v5/beacon-chain/state"

type MockExecutionPayloadHeader struct {
	ErrBuilderSlashed             error
	ErrBuilderInsufficientBalance error
	ErrUnknownParentBlockHash     error
	ErrUnknownParentBlockRoot     error
	ErrIncorrectSlot              error
	ErrInvalidSignature           error
}

var _ ExecutionPayloadHeaderVerifier = &MockExecutionPayloadHeader{}

func (e *MockExecutionPayloadHeader) VerifyBuilderActiveNotSlashed(validator state.ReadOnlyValidator) error {
	return e.ErrBuilderSlashed
}

func (e *MockExecutionPayloadHeader) VerifyBuilderSufficientBalance(uint642 uint64) error {
	return e.ErrBuilderInsufficientBalance
}

func (e *MockExecutionPayloadHeader) VerifyParentBlockHashSeen(func([32]byte) bool) error {
	return e.ErrUnknownParentBlockHash
}

func (e *MockExecutionPayloadHeader) VerifyParentBlockRootSeen(func([32]byte) bool) error {
	return e.ErrUnknownParentBlockRoot
}

func (e *MockExecutionPayloadHeader) VerifyCurrentOrNextSlot() error {
	return e.ErrIncorrectSlot
}

func (e *MockExecutionPayloadHeader) VerifySignature(validator state.ReadOnlyValidator, genesisRoot []byte) error {
	return e.ErrInvalidSignature
}

func (e *MockExecutionPayloadHeader) SatisfyRequirement(requirement Requirement) {

}
