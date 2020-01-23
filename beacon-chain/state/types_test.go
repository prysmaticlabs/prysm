package state

import (
	"strconv"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/interop"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/stateutil"
)

func TestBeaconState_ProtoBeaconStateCompatibility(t *testing.T) {
	params.UseMinimalConfig()
	genesis := setupGenesisState(t, 64)
	customState, err := InitializeFromProto(genesis)
	if err != nil {
		t.Fatal(err)
	}
	cloned := proto.Clone(genesis).(*pb.BeaconState)
	custom := customState.Clone()
	if !proto.Equal(cloned, custom) {
		t.Fatal("Cloned states did not match")
	}

	r1, err := customState.HashTreeRoot()
	if err != nil {
		t.Fatal(err)
	}
	r2, err := stateutil.HashTreeRootState(genesis)
	if err != nil {
		t.Fatal(err)
	}
	if r1 != r2 {
		t.Fatalf("Mismatched roots, custom HTR %#x != regular HTR %#x", r1, r2)
	}

	// We then write to the the state and compare hash tree roots again.
	balances := genesis.Balances
	balances[0] = 3823
	if err := customState.SetBalances(balances); err != nil {
		t.Fatal(err)
	}
	r1, err = customState.HashTreeRoot()
	if err != nil {
		t.Fatal(err)
	}
	genesis.Balances = balances
	r2, err = stateutil.HashTreeRootState(genesis)
	if err != nil {
		t.Fatal(err)
	}
	if r1 != r2 {
		t.Fatalf("Mismatched roots, custom HTR %#x != regular HTR %#x", r1, r2)
	}
}

func setupGenesisState(tb testing.TB, count uint64) *pb.BeaconState {
	genesisState, _, err := interop.GenerateGenesisState(0, count)
	if err != nil {
		tb.Fatalf("Could not generate genesis beacon state: %v", err)
	}
	for i := uint64(1); i < count; i++ {
		someRoot := [32]byte{}
		someKey := [48]byte{}
		copy(someRoot[:], strconv.Itoa(int(i)))
		copy(someKey[:], strconv.Itoa(int(i)))
		genesisState.Validators = append(genesisState.Validators, &ethpb.Validator{
			PublicKey:                  someKey[:],
			WithdrawalCredentials:      someRoot[:],
			EffectiveBalance:           params.BeaconConfig().MaxEffectiveBalance,
			Slashed:                    false,
			ActivationEligibilityEpoch: 1,
			ActivationEpoch:            1,
			ExitEpoch:                  1,
			WithdrawableEpoch:          1,
		})
		genesisState.Balances = append(genesisState.Balances, params.BeaconConfig().MaxEffectiveBalance)
	}
	return genesisState
}

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
		cloneValidatorsWithProto(validators)
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
		cloneValidatorsManually(validators)
	}
}

func BenchmarkStateClone_Proto(b *testing.B) {
	b.StopTimer()
	params.UseMinimalConfig()
	genesis := setupGenesisState(b, 64)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_ = proto.Clone(genesis).(*pb.BeaconState)
	}
}

func BenchmarkStateClone_Manual(b *testing.B) {
	b.StopTimer()
	params.UseMinimalConfig()
	genesis := setupGenesisState(b, 64)
	st, err := InitializeFromProto(genesis)
	if err != nil {
		b.Fatal(err)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_ = st.Clone()
	}
}

func cloneValidatorsWithProto(vals []*ethpb.Validator) []*ethpb.Validator {
	res := make([]*ethpb.Validator, len(vals))
	for i := 0; i < len(res); i++ {
		res[i] = proto.Clone(vals[i]).(*ethpb.Validator)
	}
	return res
}

func cloneValidatorsManually(vals []*ethpb.Validator) []*ethpb.Validator {
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
