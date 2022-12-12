package transition_test

import (
	"context"
	"sync"
	"testing"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	state_native "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/state-native"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
)

func TestSkipSlotCache_OK(t *testing.T) {
	transition.SkipSlotCache.Enable()
	defer transition.SkipSlotCache.Disable()
	bState, privs := util.DeterministicGenesisState(t, params.MinimalSpecConfig().MinGenesisActiveValidatorCount)
	bStateProto, err := bState.ToProto()
	require.NoError(t, err)
	pbState, err := state_native.ProtobufBeaconStatePhase0(bStateProto)
	require.NoError(t, err)
	originalState, err := state_native.InitializeFromProtoPhase0(pbState)
	require.NoError(t, err)

	blkCfg := util.DefaultBlockGenConfig()
	blkCfg.NumAttestations = 1

	// First transition will be with an empty cache, so the cache becomes populated
	// with the state
	blk, err := util.GenerateFullBlock(bState, privs, blkCfg, originalState.Slot()+10)
	require.NoError(t, err)
	wsb, err := blocks.NewSignedBeaconBlock(blk)
	require.NoError(t, err)
	executedState, err := transition.ExecuteStateTransition(context.Background(), originalState, wsb)
	require.NoError(t, err, "Could not run state transition")
	require.Equal(t, true, executedState.Version() == version.Phase0)
	wsb, err = blocks.NewSignedBeaconBlock(blk)
	require.NoError(t, err)
	bState, err = transition.ExecuteStateTransition(context.Background(), bState, wsb)
	require.NoError(t, err, "Could not process state transition")

	originalStateProto, err := originalState.ToProto()
	require.NoError(t, err)
	bStateProto, err = bState.ToProto()
	require.NoError(t, err)
	assert.DeepEqual(t, originalStateProto, bStateProto, "Skipped slots cache leads to different states")
}

func TestSkipSlotCache_ConcurrentMixup(t *testing.T) {
	bState, privs := util.DeterministicGenesisState(t, params.MinimalSpecConfig().MinGenesisActiveValidatorCount)
	bStateProto, err := bState.ToProto()
	require.NoError(t, err)
	pbState, err := state_native.ProtobufBeaconStatePhase0(bStateProto)
	require.NoError(t, err)
	originalState, err := state_native.InitializeFromProtoPhase0(pbState)
	require.NoError(t, err)

	blkCfg := util.DefaultBlockGenConfig()
	blkCfg.NumAttestations = 1

	transition.SkipSlotCache.Disable()

	// First transition will be with an empty cache, so the cache becomes populated
	// with the state
	blk, err := util.GenerateFullBlock(bState, privs, blkCfg, originalState.Slot()+10)
	require.NoError(t, err)
	wsb, err := blocks.NewSignedBeaconBlock(blk)
	require.NoError(t, err)
	executedState, err := transition.ExecuteStateTransition(context.Background(), originalState, wsb)
	require.NoError(t, err, "Could not run state transition")
	require.Equal(t, true, executedState.Version() == version.Phase0)

	// Create two shallow but different forks
	var s1, s0 state.BeaconState
	{
		c0, err := originalState.Copy()
		require.NoError(t, err)
		blk, err := util.GenerateFullBlock(c0, privs, blkCfg, originalState.Slot()+10)
		require.NoError(t, err)
		copy(blk.Block.Body.Graffiti, "block 1")
		signature, err := util.BlockSignature(originalState, blk.Block, privs)
		require.NoError(t, err)
		blk.Signature = signature.Marshal()
		wsb, err := blocks.NewSignedBeaconBlock(blk)
		require.NoError(t, err)
		s1, err = transition.ExecuteStateTransition(context.Background(), c0, wsb)
		require.NoError(t, err, "Could not run state transition")
	}

	{
		c1, err := originalState.Copy()
		require.NoError(t, err)
		blk, err := util.GenerateFullBlock(c1, privs, blkCfg, originalState.Slot()+10)
		require.NoError(t, err)
		copy(blk.Block.Body.Graffiti, "block 2")
		signature, err := util.BlockSignature(originalState, blk.Block, privs)
		require.NoError(t, err)
		blk.Signature = signature.Marshal()
		wsb, err := blocks.NewSignedBeaconBlock(blk)
		require.NoError(t, err)
		s0, err = transition.ExecuteStateTransition(context.Background(), c1, wsb)
		require.NoError(t, err, "Could not run state transition")
	}

	r1, err := s1.HashTreeRoot(context.Background())
	require.NoError(t, err)
	r2, err := s0.HashTreeRoot(context.Background())
	require.NoError(t, err)
	if r1 == r2 {
		t.Fatalf("need different starting states, got: %x", r1)
	}

	if s1.Slot() != s0.Slot() {
		t.Fatalf("expecting different chains, but states at same slot")
	}

	// prepare copies for both states
	var setups []state.BeaconState
	for i := uint64(0); i < 300; i++ {
		var st state.BeaconState
		if i%2 == 0 {
			st = s1
		} else {
			st = s0
		}
		c, err := st.Copy()
		require.NoError(t, err)
		setups = append(setups, c)
	}

	problemSlot := s1.Slot() + 2
	s1Copied, err := s1.Copy()
	require.NoError(t, err)
	expected1, err := transition.ProcessSlots(context.Background(), s1Copied, problemSlot)
	require.NoError(t, err)
	expectedRoot1, err := expected1.HashTreeRoot(context.Background())
	require.NoError(t, err)
	t.Logf("chain 1 (even i) expected root %x at slot %d", expectedRoot1[:], problemSlot)

	expectedS1Copied, err := expected1.Copy()
	require.NoError(t, err)
	tmp1, err := transition.ProcessSlots(context.Background(), expectedS1Copied, problemSlot+1)
	require.NoError(t, err)
	gotRoot := tmp1.StateRoots()[problemSlot]
	require.DeepEqual(t, expectedRoot1[:], gotRoot, "State roots for chain 1 are bad, expected root doesn't match")

	s0Copied, err := s0.Copy()
	require.NoError(t, err)
	expected2, err := transition.ProcessSlots(context.Background(), s0Copied, problemSlot)
	require.NoError(t, err)
	expectedRoot2, err := expected2.HashTreeRoot(context.Background())
	require.NoError(t, err)
	t.Logf("chain 2 (odd i) expected root %x at slot %d", expectedRoot2[:], problemSlot)

	expectedS2Copied, err := expected2.Copy()
	require.NoError(t, err)
	tmp2, err := transition.ProcessSlots(context.Background(), expectedS2Copied, problemSlot+1)
	require.NoError(t, err)
	gotRoot = tmp2.StateRoots()[problemSlot]
	require.DeepEqual(t, expectedRoot2[:], gotRoot, "State roots for chain 2 are bad, expected root doesn't match")

	var wg sync.WaitGroup
	wg.Add(len(setups))

	step := func(i int, setup state.BeaconState) {
		// go at least 1 past problemSlot, to ensure problem slot state root is available
		outState, err := transition.ProcessSlots(context.Background(), setup, problemSlot.Add(1+uint64(i))) // keep increasing, to hit and extend the cache
		require.NoError(t, err, "Could not process state transition")
		roots := outState.StateRoots()
		gotRoot := roots[problemSlot]
		if i%2 == 0 {
			assert.DeepEqual(t, expectedRoot1[:], gotRoot, "Unexpected root on chain 1")
		} else {
			assert.DeepEqual(t, expectedRoot2[:], gotRoot, "Unexpected root on chain 2")
		}
		wg.Done()
	}

	transition.SkipSlotCache.Enable()
	// now concurrently apply the blocks (alternating between states, and increasing skip slots)
	for i, setup := range setups {
		go step(i, setup)
	}
	// Wait for all transitions to finish
	wg.Wait()
}
