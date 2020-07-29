package cache

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestCheckpointStateCacheKeyFn_OK(t *testing.T) {
	cp := &ethpb.Checkpoint{Epoch: 1, Root: bytesutil.PadTo([]byte{'A'}, 32)}
	st, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Slot: 64,
	})
	require.NoError(t, err)

	info := &CheckpointState{
		Checkpoint: cp,
		State:      st,
	}
	key, err := checkpointState(info)
	require.NoError(t, err)

	wantedKey, err := hashutil.HashProto(cp)
	require.NoError(t, err)
	assert.Equal(t, string(wantedKey[:]), key)
}

func TestCheckpointStateCacheKeyFn_InvalidObj(t *testing.T) {
	_, err := checkpointState("bad")
	assert.Equal(t, ErrNotCheckpointState, err)
}

func TestCheckpointStateCache_StateByCheckpoint(t *testing.T) {
	cache := NewCheckpointStateCache()

	cp1 := &ethpb.Checkpoint{Epoch: 1, Root: bytesutil.PadTo([]byte{'A'}, 32)}
	st, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		GenesisValidatorsRoot: params.BeaconConfig().ZeroHash[:],
		Slot:                  64,
	})
	require.NoError(t, err)

	info1 := &CheckpointState{
		Checkpoint: cp1,
		State:      st,
	}
	state, err := cache.StateByCheckpoint(cp1)
	require.NoError(t, err)
	assert.Equal(t, (*stateTrie.BeaconState)(nil), state, "Expected state not to exist in empty cache")

	require.NoError(t, cache.AddCheckpointState(info1))

	state, err = cache.StateByCheckpoint(cp1)
	require.NoError(t, err)

	if !proto.Equal(state.InnerStateUnsafe(), info1.State.InnerStateUnsafe()) {
		t.Error("incorrectly cached state")
	}

	cp2 := &ethpb.Checkpoint{Epoch: 2, Root: bytesutil.PadTo([]byte{'B'}, 32)}
	st2, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Slot: 128,
	})
	require.NoError(t, err)

	info2 := &CheckpointState{
		Checkpoint: cp2,
		State:      st2,
	}
	require.NoError(t, cache.AddCheckpointState(info2))

	state, err = cache.StateByCheckpoint(cp2)
	require.NoError(t, err)
	assert.DeepEqual(t, info2.State.CloneInnerState(), state.CloneInnerState(), "incorrectly cached state")

	state, err = cache.StateByCheckpoint(cp1)
	require.NoError(t, err)
	assert.DeepEqual(t, info1.State.CloneInnerState(), state.CloneInnerState(), "incorrectly cached state")
}

func TestCheckpointStateCache_MaxSize(t *testing.T) {
	c := NewCheckpointStateCache()
	st, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Slot: 0,
	})
	require.NoError(t, err)

	for i := uint64(0); i < maxCheckpointStateSize+100; i++ {
		require.NoError(t, st.SetSlot(i))

		info := &CheckpointState{
			Checkpoint: &ethpb.Checkpoint{Epoch: i, Root: make([]byte, 32)},
			State:      st,
		}
		require.NoError(t, c.AddCheckpointState(info))
	}

	assert.Equal(t, maxCheckpointStateSize, uint64(len(c.cache.ListKeys())))
}
