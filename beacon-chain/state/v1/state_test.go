package v1

import (
	"context"
	"strconv"
	"sync"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/assert"
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
	handler := stateutil.NewValMapHandler(vals)
	newHandler := handler.Copy()
	wantedPubkey := strconv.Itoa(22)
	handler.Set(bytesutil.ToBytes48([]byte(wantedPubkey)), 27)
	val1, _ := handler.Get(bytesutil.ToBytes48([]byte(wantedPubkey)))
	val2, _ := newHandler.Get(bytesutil.ToBytes48([]byte(wantedPubkey)))
	assert.NotEqual(t, val1, val2, "Values are supposed to be unequal due to copy")
}

func TestBeaconState_NoDeadlock(t *testing.T) {
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
	st, err := InitializeFromProtoUnsafe(&ethpb.BeaconState{
		Validators: vals,
	})
	assert.NoError(t, err)

	wg := new(sync.WaitGroup)

	wg.Add(1)
	go func() {
		// Continuously lock and unlock the state
		// by acquiring the lock.
		for i := 0; i < 1000; i++ {
			for _, f := range st.stateFieldLeaves {
				f.Lock()
				if f.Empty() {
					f.InsertFieldLayer(make([][]*[32]byte, 10))
				}
				f.Unlock()
				f.FieldReference().AddRef()
			}
		}
		wg.Done()
	}()
	// Constantly read from the offending portion
	// of the code to ensure there is no possible
	// recursive read locking.
	for i := 0; i < 1000; i++ {
		go func() {
			_ = st.FieldReferencesCount()
		}()
	}
	// Test will not terminate in the event of a deadlock.
	wg.Wait()
}

func TestStateTrie_IsNil(t *testing.T) {
	var emptyState *BeaconState
	assert.Equal(t, true, emptyState.IsNil())

	emptyProto := &BeaconState{state: nil}
	assert.Equal(t, true, emptyProto.IsNil())

	nonNilState := &BeaconState{state: &ethpb.BeaconState{}}
	assert.Equal(t, false, nonNilState.IsNil())
}

func TestBeaconState_AppendBalanceWithTrie(t *testing.T) {
	count := uint64(100)
	vals := make([]*ethpb.Validator, 0, count)
	bals := make([]uint64, 0, count)
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
		bals = append(bals, params.BeaconConfig().MaxEffectiveBalance)
	}
	zeroHash := params.BeaconConfig().ZeroHash
	mockblockRoots := make([][]byte, params.BeaconConfig().SlotsPerHistoricalRoot)
	for i := 0; i < len(mockblockRoots); i++ {
		mockblockRoots[i] = zeroHash[:]
	}

	mockstateRoots := make([][]byte, params.BeaconConfig().SlotsPerHistoricalRoot)
	for i := 0; i < len(mockstateRoots); i++ {
		mockstateRoots[i] = zeroHash[:]
	}
	mockrandaoMixes := make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector)
	for i := 0; i < len(mockrandaoMixes); i++ {
		mockrandaoMixes[i] = zeroHash[:]
	}
	st, err := InitializeFromProto(&ethpb.BeaconState{
		Slot:                  1,
		GenesisValidatorsRoot: make([]byte, 32),
		Fork: &ethpb.Fork{
			PreviousVersion: make([]byte, 4),
			CurrentVersion:  make([]byte, 4),
			Epoch:           0,
		},
		LatestBlockHeader: &ethpb.BeaconBlockHeader{
			ParentRoot: make([]byte, 32),
			StateRoot:  make([]byte, 32),
			BodyRoot:   make([]byte, 32),
		},
		Validators: vals,
		Balances:   bals,
		Eth1Data: &ethpb.Eth1Data{
			DepositRoot: make([]byte, 32),
			BlockHash:   make([]byte, 32),
		},
		BlockRoots:                  mockblockRoots,
		StateRoots:                  mockstateRoots,
		RandaoMixes:                 mockrandaoMixes,
		JustificationBits:           bitfield.NewBitvector4(),
		PreviousJustifiedCheckpoint: &ethpb.Checkpoint{Root: make([]byte, 32)},
		CurrentJustifiedCheckpoint:  &ethpb.Checkpoint{Root: make([]byte, 32)},
		FinalizedCheckpoint:         &ethpb.Checkpoint{Root: make([]byte, 32)},
		Slashings:                   make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector),
	})
	assert.NoError(t, err)
	_, err = st.HashTreeRoot(context.Background())
	assert.NoError(t, err)

	for i := 0; i < 100; i++ {
		if i%2 == 0 {
			assert.NoError(t, st.UpdateBalancesAtIndex(types.ValidatorIndex(i), 1000))
		}
		if i%3 == 0 {
			assert.NoError(t, st.AppendBalance(1000))
		}
	}
	_, err = st.HashTreeRoot(context.Background())
	assert.NoError(t, err)
	newRt := bytesutil.ToBytes32(st.merkleLayers[0][balances])
	wantedRt, err := stateutil.Uint64ListRootWithRegistryLimit(st.state.Balances)
	assert.NoError(t, err)
	assert.Equal(t, wantedRt, newRt, "state roots are unequal")
}
