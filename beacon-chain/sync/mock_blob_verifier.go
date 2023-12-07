package sync

import (
	"context"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/verification"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
)

type BlobVerifierInitializer interface {
	NewBlobVerifier(b blocks.ROBlob, reqs ...verification.Requirement) BlobVerifier
}

type mockBlobVerifierInitializer struct{}

func (m *mockBlobVerifierInitializer) NewBlobVerifier(b blocks.ROBlob, reqs ...verification.Requirement) BlobVerifier {
	return nil
}

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

type mockBlobVerifier struct{}

func (m *mockBlobVerifier) VerifiedROBlob() (blocks.VerifiedROBlob, error) {
	return blocks.VerifiedROBlob{}, nil
}

func (m *mockBlobVerifier) BlobIndexInBounds() (err error) {
	return nil
}

func (m *mockBlobVerifier) SlotNotTooEarly() (err error) {
	return nil
}

func (m *mockBlobVerifier) SlotAboveFinalized() (err error) {
	return nil
}

func (m *mockBlobVerifier) ValidProposerSignature(ctx context.Context) (err error) {
	return nil
}

func (m *mockBlobVerifier) SidecarParentSeen(badParent func([32]byte) bool) (err error) {
	return nil
}

func (m *mockBlobVerifier) SidecarParentValid(badParent func([32]byte) bool) (err error) {
	return nil
}

func (m *mockBlobVerifier) SidecarParentSlotLower() (err error) {
	return nil
}

func (m *mockBlobVerifier) SidecarDescendsFromFinalized() (err error) {
	return nil
}

func (m *mockBlobVerifier) SidecarInclusionProven() (err error) {
	return nil
}

func (m *mockBlobVerifier) SidecarKzgProofVerified() (err error) {
	return nil
}

func (m *mockBlobVerifier) SidecarProposerExpected(ctx context.Context) (err error) {
	return nil
}
