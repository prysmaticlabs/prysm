//go:build go1.18
// +build go1.18

package v1_test

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	coreState "github.com/prysmaticlabs/prysm/beacon-chain/core/transition"
	v12 "github.com/prysmaticlabs/prysm/beacon-chain/state/state-native/v1"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/state/v1"
	"github.com/prysmaticlabs/prysm/crypto/rand"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func FuzzV1StateHashTreeRoot(f *testing.F) {
	gState, _ := util.DeterministicGenesisState(f, 100)
	output, err := gState.MarshalSSZ()
	assert.NoError(f, err)
	randPool := make([]byte, 100)
	_, err = rand.NewDeterministicGenerator().Read(randPool)
	assert.NoError(f, err)
	f.Add(randPool, uint64(10))
	f.Fuzz(func(t *testing.T, diffBuffer []byte, slotsToTransition uint64) {
		stateSSZ := bytesutil.SafeCopyBytes(output)
		for i := 0; i < len(diffBuffer); i += 9 {
			if i+8 >= len(diffBuffer) {
				return
			}
			num := bytesutil.BytesToUint64BigEndian(diffBuffer[i : i+8])
			num %= uint64(len(diffBuffer))
			// Perform a XOR on the byte of the selected index.
			stateSSZ[num] ^= diffBuffer[i+8]
		}
		pbState := &ethpb.BeaconState{}
		err := pbState.UnmarshalSSZ(stateSSZ)
		if err != nil {
			return
		}
		nativeState, err := v12.InitializeFromProto(pbState)
		assert.NoError(t, err)

		slotsToTransition %= 100
		stateObj, err := v1.InitializeFromProtoUnsafe(pbState)
		assert.NoError(t, err)
		for stateObj.Slot() < types.Slot(slotsToTransition) {
			stateObj, err = coreState.ProcessSlots(context.Background(), stateObj, stateObj.Slot()+1)
			assert.NoError(t, err)
			stateObj.Copy()

			nativeState, err = coreState.ProcessSlots(context.Background(), nativeState, nativeState.Slot()+1)
			assert.NoError(t, err)
			nativeState.Copy()
		}
		assert.NoError(t, err)
		// Perform a cold HTR calculation by initializing a new state.
		innerState := stateObj.InnerStateUnsafe().(*ethpb.BeaconState)
		newState, err := v1.InitializeFromProtoUnsafe(innerState)
		assert.NoError(t, err)

		newRt, newErr := newState.HashTreeRoot(context.Background())
		rt, err := stateObj.HashTreeRoot(context.Background())
		nativeRt, nativeErr := nativeState.HashTreeRoot(context.Background())

		assert.Equal(t, newErr != nil, err != nil)
		assert.Equal(t, newErr != nil, nativeErr != nil)
		if err == nil {
			assert.Equal(t, rt, newRt)
			assert.Equal(t, rt, nativeRt)
		}
	})
}
