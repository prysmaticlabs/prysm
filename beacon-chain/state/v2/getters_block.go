package v2

import (
	customtypes "github.com/prysmaticlabs/prysm/beacon-chain/state/custom-types"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// LatestBlockHeader stored within the beacon state.
func (b *BeaconState) LatestBlockHeader() *ethpb.BeaconBlockHeader {
	if !b.hasInnerState() {
		return nil
	}
	if b.latestBlockHeader == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.latestBlockHeaderInternal()
}

// latestBlockHeaderInternal stored within the beacon state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) latestBlockHeaderInternal() *ethpb.BeaconBlockHeader {
	if !b.hasInnerState() {
		return nil
	}
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
func (b *BeaconState) BlockRoots() *[8192][32]byte {
	if !b.hasInnerState() {
		return nil
	}
	if b.blockRoots == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	roots := [8192][32]byte(*b.blockRootsInternal())
	return &roots
}

// blockRootsInternal kept track of in the beacon state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) blockRootsInternal() *customtypes.StateRoots {
	if !b.hasInnerState() {
		return nil
	}
	return b.blockRoots
}

// BlockRootAtIndex retrieves a specific block root based on an
// input index value.
func (b *BeaconState) BlockRootAtIndex(idx uint64) ([32]byte, error) {
	if !b.hasInnerState() {
		return [32]byte{}, ErrNilInnerState
	}
	if b.blockRoots == nil {
		return [32]byte{}, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.blockRootAtIndex(idx)
}

// blockRootAtIndex retrieves a specific block root based on an
// input index value.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) blockRootAtIndex(idx uint64) ([32]byte, error) {
	if !b.hasInnerState() {
		return [32]byte{}, ErrNilInnerState
	}
	bRoots := make([][]byte, len(b.blockRoots))
	for i := range bRoots {
		tmp := b.blockRoots[i]
		bRoots[i] = tmp[:]
	}
	root, err := bytesutil.SafeCopyRootAtIndex(bRoots, idx)
	if err != nil {
		return [32]byte{}, err
	}
	return bytesutil.ToBytes32(root), nil
}
