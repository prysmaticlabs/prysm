package validator

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/gorilla/mux"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	mock "github.com/prysmaticlabs/prysm/v5/beacon-chain/blockchain/testing"
	dbTest "github.com/prysmaticlabs/prysm/v5/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/rpc/core"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/rpc/testutil"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state/stategen"
	mockstategen "github.com/prysmaticlabs/prysm/v5/beacon-chain/state/stategen/mock"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
)

func addDefaultReplayerBuilder(s *Server, h stategen.HistoryAccessor) {
	cc := &mockstategen.CanonicalChecker{Is: true, Err: nil}
	cs := &mockstategen.CurrentSlotter{Slot: math.MaxUint64 - 1}
	s.CoreService.ReplayerBuilder = stategen.NewCanonicalHistory(h, cc, cs)
}

func TestServer_GetValidatorActiveSetChanges_NoState(t *testing.T) {
	beaconDB := dbTest.SetupDB(t)
	var st state.BeaconState
	st, _ = util.DeterministicGenesisState(t, 4)

	s := &Server{
		Stater: &testutil.MockStater{
			BeaconState: st,
		},
		CoreService: &core.Service{
			BeaconDB:           beaconDB,
			GenesisTimeFetcher: &mock.ChainService{},
			HeadFetcher: &mock.ChainService{
				State: st,
			},
		},
	}

	url := "http://example.com" + fmt.Sprintf("%d", slots.ToEpoch(s.CoreService.GenesisTimeFetcher.CurrentSlot())+1)
	request := httptest.NewRequest(http.MethodGet, url, nil)
	request = mux.SetURLVars(request, map[string]string{"state_id": ""})
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.GetValidatorActiveSetChanges(writer, request)
	require.Equal(t, http.StatusBadRequest, writer.Code)
	require.StringContains(t, "state_id is required in URL params", writer.Body.String())
}

func TestServer_GetValidatorActiveSetChanges(t *testing.T) {
	beaconDB := dbTest.SetupDB(t)

	ctx := context.Background()
	validators := make([]*ethpb.Validator, 8)
	headState, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, headState.SetSlot(0))
	require.NoError(t, headState.SetValidators(validators))
	for i := 0; i < len(validators); i++ {
		activationEpoch := params.BeaconConfig().FarFutureEpoch
		withdrawableEpoch := params.BeaconConfig().FarFutureEpoch
		exitEpoch := params.BeaconConfig().FarFutureEpoch
		slashed := false
		balance := params.BeaconConfig().MaxEffectiveBalance
		// Mark indices divisible by two as activated.
		if i%2 == 0 {
			activationEpoch = 0
		} else if i%3 == 0 {
			// Mark indices divisible by 3 as slashed.
			withdrawableEpoch = params.BeaconConfig().EpochsPerSlashingsVector
			slashed = true
		} else if i%5 == 0 {
			// Mark indices divisible by 5 as exited.
			exitEpoch = 0
			withdrawableEpoch = params.BeaconConfig().MinValidatorWithdrawabilityDelay
		} else if i%7 == 0 {
			// Mark indices divisible by 7 as ejected.
			exitEpoch = 0
			withdrawableEpoch = params.BeaconConfig().MinValidatorWithdrawabilityDelay
			balance = params.BeaconConfig().EjectionBalance
		}
		err := headState.UpdateValidatorAtIndex(primitives.ValidatorIndex(i), &ethpb.Validator{
			ActivationEpoch:       activationEpoch,
			PublicKey:             pubKey(uint64(i)),
			EffectiveBalance:      balance,
			WithdrawalCredentials: make([]byte, 32),
			WithdrawableEpoch:     withdrawableEpoch,
			Slashed:               slashed,
			ExitEpoch:             exitEpoch,
		})
		require.NoError(t, err)
	}
	b := util.NewBeaconBlock()
	util.SaveBlock(t, ctx, beaconDB, b)

	gRoot, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, gRoot))
	require.NoError(t, beaconDB.SaveState(ctx, headState, gRoot))

	var st state.BeaconState
	st, _ = util.DeterministicGenesisState(t, 4)
	s := &Server{
		Stater: &testutil.MockStater{
			BeaconState: st,
		},
		CoreService: &core.Service{
			FinalizedFetcher: &mock.ChainService{
				FinalizedCheckPoint: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, fieldparams.RootLength)},
			},
			GenesisTimeFetcher: &mock.ChainService{},
		},
	}
	addDefaultReplayerBuilder(s, beaconDB)

	url := "http://example.com"
	request := httptest.NewRequest(http.MethodGet, url, nil)
	request = mux.SetURLVars(request, map[string]string{"state_id": "genesis"})
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.GetValidatorActiveSetChanges(writer, request)
	require.Equal(t, http.StatusOK, writer.Code)

	wantedActive := []string{
		hexutil.Encode(pubKey(0)),
		hexutil.Encode(pubKey(2)),
		hexutil.Encode(pubKey(4)),
		hexutil.Encode(pubKey(6)),
	}
	wantedActiveIndices := []string{"0", "2", "4", "6"}
	wantedExited := []string{
		hexutil.Encode(pubKey(5)),
	}
	wantedExitedIndices := []string{"5"}
	wantedSlashed := []string{
		hexutil.Encode(pubKey(3)),
	}
	wantedSlashedIndices := []string{"3"}
	wantedEjected := []string{
		hexutil.Encode(pubKey(7)),
	}
	wantedEjectedIndices := []string{"7"}
	want := &structs.ActiveSetChanges{
		Epoch:               "0",
		ActivatedPublicKeys: wantedActive,
		ActivatedIndices:    wantedActiveIndices,
		ExitedPublicKeys:    wantedExited,
		ExitedIndices:       wantedExitedIndices,
		SlashedPublicKeys:   wantedSlashed,
		SlashedIndices:      wantedSlashedIndices,
		EjectedPublicKeys:   wantedEjected,
		EjectedIndices:      wantedEjectedIndices,
	}

	var as *structs.ActiveSetChanges
	err = json.NewDecoder(writer.Body).Decode(&as)
	require.NoError(t, err)
	require.DeepEqual(t, *want, *as)
}

func pubKey(i uint64) []byte {
	pubKey := make([]byte, params.BeaconConfig().BLSPubkeyLength)
	binary.LittleEndian.PutUint64(pubKey, i)
	return pubKey
}
