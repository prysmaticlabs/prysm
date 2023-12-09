package sync

import (
	"context"

	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
)

type BlobVerifier interface {
	VerifiedROBlob() (blocks.VerifiedROBlob, error)
	BlobIndexInBounds() (err error)
	SlotNotTooEarly() (err error)
	SlotAboveFinalized() (err error)
	ValidProposerSignature(ctx context.Context) (err error)
	SidecarParentSeen(badParent func([32]byte) bool) (err error)
	SidecarParentValid(badParent func([32]byte) bool) (err error)
	SidecarParentSlotLower() (err error)
	SidecarDescendsFromFinalized() (err error)
	SidecarInclusionProven() (err error)
	SidecarKzgProofVerified() (err error)
	SidecarProposerExpected(ctx context.Context) (err error)
}

type mockBlobVerifier struct {
	errBlobIndexInBounds            error
	errSlotTooEarly                 error
	errSlotAboveFinalized           error
	errValidProposerSignature       error
	errSidecarParentSeen            error
	errSidecarParentValid           error
	errSidecarParentSlotLower       error
	errSidecarDescendsFromFinalized error
	errSidecarInclusionProven       error
	errSidecarKzgProofVerified      error
	errSidecarProposerExpected      error
}

func (m *mockBlobVerifier) VerifiedROBlob() (blocks.VerifiedROBlob, error) {
	return blocks.VerifiedROBlob{}, nil
}

func (m *mockBlobVerifier) BlobIndexInBounds() (err error) {
	return m.errBlobIndexInBounds
}

func (m *mockBlobVerifier) SlotNotTooEarly() (err error) {
	return m.errSlotTooEarly
}

func (m *mockBlobVerifier) SlotAboveFinalized() (err error) {
	return m.errSlotAboveFinalized
}

func (m *mockBlobVerifier) ValidProposerSignature(ctx context.Context) (err error) {
	return m.errValidProposerSignature
}

func (m *mockBlobVerifier) SidecarParentSeen(badParent func([32]byte) bool) (err error) {
	return m.errSidecarParentSeen
}

func (m *mockBlobVerifier) SidecarParentValid(badParent func([32]byte) bool) (err error) {
	return m.errSidecarParentValid
}

func (m *mockBlobVerifier) SidecarParentSlotLower() (err error) {
	return m.errSidecarParentSlotLower
}

func (m *mockBlobVerifier) SidecarDescendsFromFinalized() (err error) {
	return m.errSidecarDescendsFromFinalized
}

func (m *mockBlobVerifier) SidecarInclusionProven() (err error) {
	return m.errSidecarInclusionProven
}

func (m *mockBlobVerifier) SidecarKzgProofVerified() (err error) {
	return m.errSidecarKzgProofVerified
}

func (m *mockBlobVerifier) SidecarProposerExpected(ctx context.Context) (err error) {
	return m.errSidecarProposerExpected
}
