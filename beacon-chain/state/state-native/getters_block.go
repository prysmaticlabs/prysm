package state_native

import (
	customtypes "github.com/prysmaticlabs/prysm/v4/beacon-chain/state/state-native/custom-types"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

// LatestBlockHeader stored within the beacon state.
func (b *BeaconState) LatestBlockHeader() *ethpb.BeaconBlockHeader {
	if b.latestBlockHeader == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.latestBlockHeaderVal()
}

// latestBlockHeaderVal stored within the beacon state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) latestBlockHeaderVal() *ethpb.BeaconBlockHeader {
	if b.latestBlockHeader == nil {
		return nil
	}

	hdr := &ethpb.BeaconBlockHeader{
		Slot:          b.latestBlockHeader.Slot,
		ProposerIndex: b.latestBlockHeader.ProposerIndex,
	}

	parentRoot := make([]byte, len(b.latestBlockHeader.ParentRoot))
	bodyRoot := make([]byte, len(b.latestBlockHeader.BodyRoot))
	stateRoot := make([]byte, len(b.latestBlockHeader.StateRoot))

	copy(parentRoot, b.latestBlockHeader.ParentRoot)
	copy(bodyRoot, b.latestBlockHeader.BodyRoot)
	copy(stateRoot, b.latestBlockHeader.StateRoot)
	hdr.ParentRoot = parentRoot
	hdr.BodyRoot = bodyRoot
	hdr.StateRoot = stateRoot
	return hdr
}

// BlockRoots kept track of in the beacon state.
func (b *BeaconState) BlockRoots() [][]byte {
	if b.blockRoots == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	roots := customtypes.BlockRoots(b.blockRoots.Value(b))
	return roots.Slice()
}

// BlockRootAtIndex retrieves a specific block root based on an
// input index value.
func (b *BeaconState) BlockRootAtIndex(idx uint64) ([]byte, error) {
	if b.blockRoots == nil {
		return []byte{}, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	r, err := b.blockRoots.At(b, idx)
	if err != nil {
		return nil, err
	}
	return r[:], nil
}
