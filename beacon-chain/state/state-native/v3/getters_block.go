package v3

import (
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// LatestBlockHeader stored within the beacon state.
func (b *BeaconState) LatestBlockHeader() *ethpb.BeaconBlockHeader {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.LatestBlockHeader == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.latestBlockHeader()
}

// latestBlockHeader stored within the beacon state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) latestBlockHeader() *ethpb.BeaconBlockHeader {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.LatestBlockHeader == nil {
		return nil
	}

	hdr := &ethpb.BeaconBlockHeader{
		Slot:          b.state.LatestBlockHeader.Slot,
		ProposerIndex: b.state.LatestBlockHeader.ProposerIndex,
	}

	parentRoot := make([]byte, len(b.state.LatestBlockHeader.ParentRoot))
	bodyRoot := make([]byte, len(b.state.LatestBlockHeader.BodyRoot))
	stateRoot := make([]byte, len(b.state.LatestBlockHeader.StateRoot))

	copy(parentRoot, b.state.LatestBlockHeader.ParentRoot)
	copy(bodyRoot, b.state.LatestBlockHeader.BodyRoot)
	copy(stateRoot, b.state.LatestBlockHeader.StateRoot)
	hdr.ParentRoot = parentRoot
	hdr.BodyRoot = bodyRoot
	hdr.StateRoot = stateRoot
	return hdr
}

// BlockRoots kept track of in the beacon state.
func (b *BeaconState) BlockRoots() [][]byte {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.BlockRoots == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.blockRoots()
}

// blockRoots kept track of in the beacon state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) blockRoots() [][]byte {
	if !b.hasInnerState() {
		return nil
	}
	return bytesutil.SafeCopy2dBytes(b.state.BlockRoots)
}

// BlockRootAtIndex retrieves a specific block root based on an
// input index value.
func (b *BeaconState) BlockRootAtIndex(idx uint64) ([]byte, error) {
	if !b.hasInnerState() {
		return nil, ErrNilInnerState
	}
	if b.state.BlockRoots == nil {
		return nil, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.blockRootAtIndex(idx)
}

// blockRootAtIndex retrieves a specific block root based on an
// input index value.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) blockRootAtIndex(idx uint64) ([]byte, error) {
	if !b.hasInnerState() {
		return nil, ErrNilInnerState
	}
	return bytesutil.SafeCopyRootAtIndex(b.state.BlockRoots, idx)
}
