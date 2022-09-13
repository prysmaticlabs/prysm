package transition

import (
	"context"
	"errors"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
)

// SkipSlotCache exists for the unlikely scenario that is a large gap between the head state and
// the current slot. If the beacon chain were ever to be stalled for several epochs, it may be
// difficult or impossible to compute the appropriate beacon state for assignments within a
// reasonable amount of time.
var SkipSlotCache = cache.NewSkipSlotCache()

// SkipSlotCacheKey is the key for skip slot cache is mixed between state root and state slot.
// state root is in the mix to defend against different forks with same skip slots
// to hit the same cache. We don't want beacon states mixed up between different chains.
// [0:24] represents the state root
// [24:32] represents the state slot
func SkipSlotCacheKey(_ context.Context, state state.ReadOnlyBeaconState) ([32]byte, error) {
	bh := state.LatestBlockHeader()
	if bh == nil {
		return [32]byte{}, errors.New("block head in state can't be nil")
	}
	sr := bh.StateRoot
	if len(sr) != 32 {
		return [32]byte{}, errors.New("invalid state root in latest block header")
	}

	var b [8]byte
	copy(b[:], bytesutil.SlotToBytesBigEndian(state.Slot()))
	sr = append(sr[:24], b[:]...)
	return bytesutil.ToBytes32(sr), nil
}
