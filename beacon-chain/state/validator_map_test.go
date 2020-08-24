package state

import (
	"strconv"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
)

func TestValidatorMap_DistinctCopy(t *testing.T) {
	count := uint64(100)
	vals := make([]*ethpb.Validator, 0, count)
	for i := uint64(1); i < count; i++ {
		someRoot := [32]byte{}
		someKey := [48]byte{}
		copy(someRoot[:], strconv.Itoa(int(i)))
		copy(someKey[:], strconv.Itoa(int(i)))
		vals = append(vals, &ethpb.Validator{
			PublicKey:                  someKey[:],
			WithdrawalCredentials:      someRoot[:],
			EffectiveBalance:           params.BeaconConfig().MaxEffectiveBalance,
			Slashed:                    false,
			ActivationEligibilityEpoch: 1,
			ActivationEpoch:            1,
			ExitEpoch:                  1,
			WithdrawableEpoch:          1,
		})
	}
	handler := newValHandler(vals)
	newHandler := handler.copy()
	wantedPubkey := strconv.Itoa(22)
	handler.valIdxMap[bytesutil.ToBytes48([]byte(wantedPubkey))] = 27
	assert.NotEqual(t, handler.valIdxMap[bytesutil.ToBytes48([]byte(wantedPubkey))], newHandler.valIdxMap[bytesutil.ToBytes48([]byte(wantedPubkey))], "Values are supposed to be unequal due to copy")
}
