package state

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func BenchmarkCloneValidators_Proto(b *testing.B) {
	b.StopTimer()
	validators := make([]*ethpb.Validator, 16384)
	somePubKey := [48]byte{1, 2, 3}
	someRoot := [32]byte{3, 4, 5}
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			PublicKey:                  somePubKey[:],
			WithdrawalCredentials:      someRoot[:],
			EffectiveBalance:           params.BeaconConfig().MaxEffectiveBalance,
			Slashed:                    false,
			ActivationEligibilityEpoch: params.BeaconConfig().FarFutureEpoch,
			ActivationEpoch:            3,
			ExitEpoch:                  4,
			WithdrawableEpoch:          5,
		}
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		cloneAll(validators)
	}
}

func BenchmarkCloneValidators_Manual(b *testing.B) {
	b.StopTimer()
	validators := make([]*ethpb.Validator, 16384)
	somePubKey := [48]byte{1, 2, 3}
	someRoot := [32]byte{3, 4, 5}
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			PublicKey:                  somePubKey[:],
			WithdrawalCredentials:      someRoot[:],
			EffectiveBalance:           params.BeaconConfig().MaxEffectiveBalance,
			Slashed:                    false,
			ActivationEligibilityEpoch: params.BeaconConfig().FarFutureEpoch,
			ActivationEpoch:            3,
			ExitEpoch:                  4,
			WithdrawableEpoch:          5,
		}
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		manualCopy(validators)
	}
}

func cloneAll(vals []*ethpb.Validator) []*ethpb.Validator {
	res := make([]*ethpb.Validator, len(vals))
	for i := 0; i < len(res); i++ {
		res[i] = proto.Clone(vals[i]).(*ethpb.Validator)
	}
	return res
}

func manualCopy(vals []*ethpb.Validator) []*ethpb.Validator {
	res := make([]*ethpb.Validator, len(vals))
	for i := 0; i < len(res); i++ {
		val := vals[i]
		res[i] = &ethpb.Validator{
			PublicKey:                  val.PublicKey,
			WithdrawalCredentials:      val.WithdrawalCredentials,
			EffectiveBalance:           val.EffectiveBalance,
			Slashed:                    val.Slashed,
			ActivationEligibilityEpoch: val.ActivationEligibilityEpoch,
			ActivationEpoch:            val.ActivationEpoch,
			ExitEpoch:                  val.ExitEpoch,
			WithdrawableEpoch:          val.WithdrawableEpoch,
		}
	}
	return res
}
