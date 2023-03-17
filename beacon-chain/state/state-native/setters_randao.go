package state_native

import (
	"github.com/pkg/errors"
	customtypes "github.com/prysmaticlabs/prysm/v4/beacon-chain/state/state-native/custom-types"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state/state-native/types"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state/stateutil"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
)

// SetRandaoMixes for the beacon state. Updates the entire
// randao mixes to a new value by overwriting the previous one.
func (b *BeaconState) SetRandaoMixes(val [][]byte) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sharedFieldReferences[types.RandaoMixes].MinusRef()
	b.sharedFieldReferences[types.RandaoMixes] = stateutil.NewRef(1)

	var mixesArr [fieldparams.RandaoMixesLength][32]byte
	for i := 0; i < len(mixesArr); i++ {
		copy(mixesArr[i][:], val[i])
	}
	mixes := customtypes.RandaoMixes(mixesArr)
	b.randaoMixes = &mixes
	b.markFieldAsDirty(types.RandaoMixes)
	b.rebuildTrie[types.RandaoMixes] = true
	return nil
}

// UpdateRandaoMixesAtIndex for the beacon state. Updates the randao mixes
// at a specific index to a new value.
func (b *BeaconState) UpdateRandaoMixesAtIndex(idx uint64, val []byte) error {
	if uint64(len(b.randaoMixes)) <= idx {
		return errors.Errorf("invalid index provided %d", idx)
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	mixes := b.randaoMixes
	if refs := b.sharedFieldReferences[types.RandaoMixes].Refs(); refs > 1 {
		// Copy elements in underlying array by reference.
		m := *b.randaoMixes
		mCopy := m
		mixes = &mCopy
		b.sharedFieldReferences[types.RandaoMixes].MinusRef()
		b.sharedFieldReferences[types.RandaoMixes] = stateutil.NewRef(1)
	}

	mixes[idx] = bytesutil.ToBytes32(val)
	b.randaoMixes = mixes
	b.markFieldAsDirty(types.RandaoMixes)
	b.addDirtyIndices(types.RandaoMixes, []uint64{idx})

	return nil
}
