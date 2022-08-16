//go:build go1.18

package v1_test

import (
	"context"
	"encoding/base64"
	"testing"

	coreState "github.com/prysmaticlabs/prysm/v3/beacon-chain/core/transition"
	native "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/state-native"
	v1 "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/v1"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/crypto/rand"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
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
		nativeState, err := native.InitializeFromProtoPhase0(pbState)
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
		innerState, ok := stateObj.InnerStateUnsafe().(*ethpb.BeaconState)
		assert.Equal(t, true, ok, "inner state is a not a beacon state proto")
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

		newSSZ, newErr := newState.MarshalSSZ()
		stateObjSSZ, err := stateObj.MarshalSSZ()
		nativeSSZ, nativeErr := nativeState.MarshalSSZ()
		assert.Equal(t, newErr != nil, err != nil)
		assert.Equal(t, newErr != nil, nativeErr != nil)
		if err == nil {
			assert.DeepEqual(t, newSSZ, stateObjSSZ)
			assert.DeepEqual(t, newSSZ, nativeSSZ)
		}
	})
}

func FuzzV1StateUnmarshalSSZ(f *testing.F) {
	// See example in https://github.com/prysmaticlabs/prysm/issues/5167
	b, err := base64.StdEncoding.DecodeString("AgAGAAAA5AAAAAAAAAAAAAAAAAAAAAAAAAB51og2NJR6COTeAGdBDUL2wythbDd/ntOHzD/JAg6kywEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAHnWiDY0lHoI5N4AZ0ENQvbDK2FsN3+e04fMP8kCDqTLti/UWQrIlR5nKGy5okElUBusCaDxo+6f0kX7B54ry9byYg5qzCruPLf/SccXww14BppwTrLrI/CqCbt6AWTO4kS/jz0no8RYv4wBhofviY9/W28LdzMhh6XVRUzK2Us4/wE=")
	require.NoError(f, err)
	f.Add(b)

	f.Fuzz(func(t *testing.T, b []byte) {
		pbState := &ethpb.BeaconState{}
		if err := pbState.UnmarshalSSZ(b); err != nil {
			return // Do nothing
		}
	})
}
