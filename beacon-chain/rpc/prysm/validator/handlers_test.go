package validator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	mock "github.com/prysmaticlabs/prysm/v5/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/transition"
	dbTest "github.com/prysmaticlabs/prysm/v5/beacon-chain/db/testing"
	doublylinkedtree "github.com/prysmaticlabs/prysm/v5/beacon-chain/forkchoice/doubly-linked-tree"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/rpc/core"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state/stategen"
	mockstategen "github.com/prysmaticlabs/prysm/v5/beacon-chain/state/stategen/mock"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	blocktest "github.com/prysmaticlabs/prysm/v5/consensus-types/blocks/testing"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
	prysmTime "github.com/prysmaticlabs/prysm/v5/time"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
)

func addDefaultReplayerBuilder(s *Server, h stategen.HistoryAccessor) {
	cc := &mockstategen.CanonicalChecker{Is: true, Err: nil}
	cs := &mockstategen.CurrentSlotter{Slot: math.MaxUint64 - 1}
	s.CoreService.ReplayerBuilder = stategen.NewCanonicalHistory(h, cc, cs)
}

func TestServer_GetValidatorParticipation_CannotRequestFutureEpoch(t *testing.T) {
	headState, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, headState.SetSlot(0))
	s := &Server{
		CoreService: &core.Service{
			HeadFetcher: &mock.ChainService{
				State: headState,
			},
			GenesisTimeFetcher: &mock.ChainService{},
		},
	}

	url := "http://example.com?epoch=" + fmt.Sprintf("%d", slots.ToEpoch(s.CoreService.GenesisTimeFetcher.CurrentSlot())+1)
	request := httptest.NewRequest(http.MethodGet, url, nil)
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.GetValidatorParticipation(writer, request)
	require.Equal(t, http.StatusBadRequest, writer.Code)
	require.StringContains(t, "cannot retrieve information about an epoch", writer.Body.String())
}

func TestServer_GetValidatorParticipation_CurrentAndPrevEpoch(t *testing.T) {
	helpers.ClearCache()
	beaconDB := dbTest.SetupDB(t)

	ctx := context.Background()
	validatorCount := uint64(32)

	validators := make([]*ethpb.Validator, validatorCount)
	balances := make([]uint64, validatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			PublicKey:             bytesutil.ToBytes(uint64(i), 48),
			WithdrawalCredentials: make([]byte, 32),
			ExitEpoch:             params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance:      params.BeaconConfig().MaxEffectiveBalance,
		}
		balances[i] = params.BeaconConfig().MaxEffectiveBalance
	}

	atts := []*ethpb.PendingAttestation{{
		Data:            util.HydrateAttestationData(&ethpb.AttestationData{}),
		InclusionDelay:  1,
		AggregationBits: bitfield.NewBitlist(validatorCount / uint64(params.BeaconConfig().SlotsPerEpoch)),
	}}
	headState, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, headState.SetSlot(8))
	require.NoError(t, headState.SetValidators(validators))
	require.NoError(t, headState.SetBalances(balances))
	require.NoError(t, headState.AppendCurrentEpochAttestations(atts[0]))
	require.NoError(t, headState.AppendPreviousEpochAttestations(atts[0]))

	b := util.NewBeaconBlock()
	b.Block.Slot = 8
	util.SaveBlock(t, ctx, beaconDB, b)
	bRoot, err := b.Block.HashTreeRoot()
	require.NoError(t, beaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{Root: bRoot[:]}))
	require.NoError(t, beaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{Root: params.BeaconConfig().ZeroHash[:]}))
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, bRoot))
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveState(ctx, headState, bRoot))
	require.NoError(t, beaconDB.SaveState(ctx, headState, params.BeaconConfig().ZeroHash))

	m := &mock.ChainService{State: headState}
	offset := int64(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().SecondsPerSlot))
	s := &Server{
		BeaconDB: beaconDB,
		CoreService: &core.Service{
			HeadFetcher: m,
			StateGen:    stategen.New(beaconDB, doublylinkedtree.New()),
			GenesisTimeFetcher: &mock.ChainService{
				Genesis: prysmTime.Now().Add(time.Duration(-1*offset) * time.Second),
			},
			FinalizedFetcher: &mock.ChainService{FinalizedCheckPoint: &ethpb.Checkpoint{Epoch: 100}},
		},
		CanonicalFetcher: &mock.ChainService{
			CanonicalRoots: map[[32]byte]bool{
				bRoot: true,
			},
		},
	}
	addDefaultReplayerBuilder(s, beaconDB)

	url := "http://example.com?epoch=1"
	request := httptest.NewRequest(http.MethodGet, url, nil)
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.GetValidatorParticipation(writer, request)
	assert.Equal(t, http.StatusOK, writer.Code)

	want := &structs.GetValidatorParticipationResponse{
		Participation: &structs.ValidatorParticipation{
			GlobalParticipationRate:          fmt.Sprintf("%f", float32(params.BeaconConfig().EffectiveBalanceIncrement)/float32(validatorCount*params.BeaconConfig().MaxEffectiveBalance)),
			VotedEther:                       fmt.Sprintf("%d", params.BeaconConfig().EffectiveBalanceIncrement),
			EligibleEther:                    fmt.Sprintf("%d", validatorCount*params.BeaconConfig().MaxEffectiveBalance),
			CurrentEpochActiveGwei:           fmt.Sprintf("%d", validatorCount*params.BeaconConfig().MaxEffectiveBalance),
			CurrentEpochAttestingGwei:        fmt.Sprintf("%d", params.BeaconConfig().EffectiveBalanceIncrement),
			CurrentEpochTargetAttestingGwei:  fmt.Sprintf("%d", params.BeaconConfig().EffectiveBalanceIncrement),
			PreviousEpochActiveGwei:          fmt.Sprintf("%d", validatorCount*params.BeaconConfig().MaxEffectiveBalance),
			PreviousEpochAttestingGwei:       fmt.Sprintf("%d", params.BeaconConfig().EffectiveBalanceIncrement),
			PreviousEpochTargetAttestingGwei: fmt.Sprintf("%d", params.BeaconConfig().EffectiveBalanceIncrement),
			PreviousEpochHeadAttestingGwei:   fmt.Sprintf("%d", params.BeaconConfig().EffectiveBalanceIncrement),
		},
	}
	var vp *structs.GetValidatorParticipationResponse
	err = json.NewDecoder(writer.Body).Decode(&vp)
	require.NoError(t, err)

	// Compare the response with the expected values
	assert.Equal(t, true, vp.Finalized, "Incorrect validator participation response")
	assert.Equal(t, *want.Participation, *vp.Participation, "Incorrect validator participation response")
}

func TestServer_GetValidatorParticipation_OrphanedUntilGenesis(t *testing.T) {
	helpers.ClearCache()
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.BeaconConfig())

	beaconDB := dbTest.SetupDB(t)
	ctx := context.Background()
	validatorCount := uint64(100)

	validators := make([]*ethpb.Validator, validatorCount)
	balances := make([]uint64, validatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			PublicKey:             bytesutil.ToBytes(uint64(i), 48),
			WithdrawalCredentials: make([]byte, 32),
			ExitEpoch:             params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance:      params.BeaconConfig().MaxEffectiveBalance,
		}
		balances[i] = params.BeaconConfig().MaxEffectiveBalance
	}

	atts := []*ethpb.PendingAttestation{{
		Data:            util.HydrateAttestationData(&ethpb.AttestationData{}),
		InclusionDelay:  1,
		AggregationBits: bitfield.NewBitlist((validatorCount / 3) / uint64(params.BeaconConfig().SlotsPerEpoch)),
	}}
	headState, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, headState.SetSlot(0))
	require.NoError(t, headState.SetValidators(validators))
	require.NoError(t, headState.SetBalances(balances))
	require.NoError(t, headState.AppendCurrentEpochAttestations(atts[0]))
	require.NoError(t, headState.AppendPreviousEpochAttestations(atts[0]))

	b := util.NewBeaconBlock()
	util.SaveBlock(t, ctx, beaconDB, b)
	bRoot, err := b.Block.HashTreeRoot()
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, bRoot))
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveState(ctx, headState, bRoot))
	require.NoError(t, beaconDB.SaveState(ctx, headState, params.BeaconConfig().ZeroHash))

	m := &mock.ChainService{State: headState}
	offset := int64(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().SecondsPerSlot))
	s := &Server{
		BeaconDB: beaconDB,
		CoreService: &core.Service{
			HeadFetcher: m,
			StateGen:    stategen.New(beaconDB, doublylinkedtree.New()),
			GenesisTimeFetcher: &mock.ChainService{
				Genesis: prysmTime.Now().Add(time.Duration(-1*offset) * time.Second),
			},
			FinalizedFetcher: &mock.ChainService{FinalizedCheckPoint: &ethpb.Checkpoint{Epoch: 100}},
			CanonicalFetcher: &mock.ChainService{
				CanonicalRoots: map[[32]byte]bool{
					bRoot: true,
				},
			},
		},
	}
	addDefaultReplayerBuilder(s, beaconDB)

	url := "http://example.com?epoch=1"
	request := httptest.NewRequest(http.MethodGet, url, nil)
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.GetValidatorParticipation(writer, request)
	assert.Equal(t, http.StatusOK, writer.Code)

	want := &structs.GetValidatorParticipationResponse{
		Participation: &structs.ValidatorParticipation{
			GlobalParticipationRate:          fmt.Sprintf("%f", float32(params.BeaconConfig().EffectiveBalanceIncrement)/float32(validatorCount*params.BeaconConfig().MaxEffectiveBalance)),
			VotedEther:                       fmt.Sprintf("%d", params.BeaconConfig().EffectiveBalanceIncrement),
			EligibleEther:                    fmt.Sprintf("%d", validatorCount*params.BeaconConfig().MaxEffectiveBalance),
			CurrentEpochActiveGwei:           fmt.Sprintf("%d", validatorCount*params.BeaconConfig().MaxEffectiveBalance),
			CurrentEpochAttestingGwei:        fmt.Sprintf("%d", params.BeaconConfig().EffectiveBalanceIncrement),
			CurrentEpochTargetAttestingGwei:  fmt.Sprintf("%d", params.BeaconConfig().EffectiveBalanceIncrement),
			PreviousEpochActiveGwei:          fmt.Sprintf("%d", validatorCount*params.BeaconConfig().MaxEffectiveBalance),
			PreviousEpochAttestingGwei:       fmt.Sprintf("%d", params.BeaconConfig().EffectiveBalanceIncrement),
			PreviousEpochTargetAttestingGwei: fmt.Sprintf("%d", params.BeaconConfig().EffectiveBalanceIncrement),
			PreviousEpochHeadAttestingGwei:   fmt.Sprintf("%d", params.BeaconConfig().EffectiveBalanceIncrement),
		},
	}
	var vp *structs.GetValidatorParticipationResponse
	err = json.NewDecoder(writer.Body).Decode(&vp)
	require.NoError(t, err)

	assert.DeepEqual(t, true, vp.Finalized, "Incorrect validator participation respond")
	assert.DeepEqual(t, *want.Participation, *vp.Participation, "Incorrect validator participation respond")
}

func TestServer_GetValidatorParticipation_CurrentAndPrevEpochWithBits(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.BeaconConfig())
	transition.SkipSlotCache.Disable()

	t.Run("altair", func(t *testing.T) {
		validatorCount := uint64(32)
		genState, _ := util.DeterministicGenesisStateAltair(t, validatorCount)
		c, err := altair.NextSyncCommittee(context.Background(), genState)
		require.NoError(t, err)
		require.NoError(t, genState.SetCurrentSyncCommittee(c))

		bits := make([]byte, validatorCount)
		for i := range bits {
			bits[i] = 0xff
		}
		require.NoError(t, genState.SetCurrentParticipationBits(bits))
		require.NoError(t, genState.SetPreviousParticipationBits(bits))
		gb, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlockAltair())
		assert.NoError(t, err)
		runGetValidatorParticipationCurrentAndPrevEpoch(t, genState, gb)
	})

	t.Run("bellatrix", func(t *testing.T) {
		validatorCount := uint64(32)
		genState, _ := util.DeterministicGenesisStateBellatrix(t, validatorCount)
		c, err := altair.NextSyncCommittee(context.Background(), genState)
		require.NoError(t, err)
		require.NoError(t, genState.SetCurrentSyncCommittee(c))

		bits := make([]byte, validatorCount)
		for i := range bits {
			bits[i] = 0xff
		}
		require.NoError(t, genState.SetCurrentParticipationBits(bits))
		require.NoError(t, genState.SetPreviousParticipationBits(bits))
		gb, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlockBellatrix())
		assert.NoError(t, err)
		runGetValidatorParticipationCurrentAndPrevEpoch(t, genState, gb)
	})

	t.Run("capella", func(t *testing.T) {
		validatorCount := uint64(32)
		genState, _ := util.DeterministicGenesisStateCapella(t, validatorCount)
		c, err := altair.NextSyncCommittee(context.Background(), genState)
		require.NoError(t, err)
		require.NoError(t, genState.SetCurrentSyncCommittee(c))

		bits := make([]byte, validatorCount)
		for i := range bits {
			bits[i] = 0xff
		}
		require.NoError(t, genState.SetCurrentParticipationBits(bits))
		require.NoError(t, genState.SetPreviousParticipationBits(bits))
		gb, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlockCapella())
		assert.NoError(t, err)
		runGetValidatorParticipationCurrentAndPrevEpoch(t, genState, gb)
	})
}

func runGetValidatorParticipationCurrentAndPrevEpoch(t *testing.T, genState state.BeaconState, gb interfaces.SignedBeaconBlock) {
	helpers.ClearCache()
	beaconDB := dbTest.SetupDB(t)

	ctx := context.Background()
	validatorCount := uint64(32)

	gsr, err := genState.HashTreeRoot(ctx)
	require.NoError(t, err)
	gb, err = blocktest.SetBlockStateRoot(gb, gsr)
	require.NoError(t, err)
	require.NoError(t, err)
	gRoot, err := gb.Block().HashTreeRoot()
	require.NoError(t, err)

	require.NoError(t, beaconDB.SaveState(ctx, genState, gRoot))
	require.NoError(t, beaconDB.SaveBlock(ctx, gb))
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, gRoot))

	m := &mock.ChainService{State: genState}
	offset := int64(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().SecondsPerSlot))
	s := &Server{
		BeaconDB: beaconDB,
		CoreService: &core.Service{
			HeadFetcher: m,
			StateGen:    stategen.New(beaconDB, doublylinkedtree.New()),
			GenesisTimeFetcher: &mock.ChainService{
				Genesis: prysmTime.Now().Add(time.Duration(-1*offset) * time.Second),
			},
			FinalizedFetcher: &mock.ChainService{FinalizedCheckPoint: &ethpb.Checkpoint{Epoch: 100}},
		},
	}
	addDefaultReplayerBuilder(s, beaconDB)

	url := "http://example.com?epoch=0"
	request := httptest.NewRequest(http.MethodGet, url, nil)
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.GetValidatorParticipation(writer, request)
	assert.Equal(t, http.StatusOK, writer.Code)

	want := &structs.GetValidatorParticipationResponse{
		Participation: &structs.ValidatorParticipation{
			GlobalParticipationRate:          "1.000000",
			VotedEther:                       fmt.Sprintf("%d", validatorCount*params.BeaconConfig().MaxEffectiveBalance),
			EligibleEther:                    fmt.Sprintf("%d", validatorCount*params.BeaconConfig().MaxEffectiveBalance),
			CurrentEpochActiveGwei:           fmt.Sprintf("%d", validatorCount*params.BeaconConfig().MaxEffectiveBalance),
			CurrentEpochAttestingGwei:        fmt.Sprintf("%d", validatorCount*params.BeaconConfig().MaxEffectiveBalance),
			CurrentEpochTargetAttestingGwei:  fmt.Sprintf("%d", validatorCount*params.BeaconConfig().MaxEffectiveBalance),
			PreviousEpochActiveGwei:          fmt.Sprintf("%d", validatorCount*params.BeaconConfig().MaxEffectiveBalance),
			PreviousEpochAttestingGwei:       fmt.Sprintf("%d", validatorCount*params.BeaconConfig().MaxEffectiveBalance),
			PreviousEpochTargetAttestingGwei: fmt.Sprintf("%d", validatorCount*params.BeaconConfig().MaxEffectiveBalance),
			PreviousEpochHeadAttestingGwei:   fmt.Sprintf("%d", validatorCount*params.BeaconConfig().MaxEffectiveBalance),
		},
	}

	var vp *structs.GetValidatorParticipationResponse
	err = json.NewDecoder(writer.Body).Decode(&vp)
	require.NoError(t, err)

	assert.DeepEqual(t, true, vp.Finalized, "Incorrect validator participation respond")
	assert.DeepEqual(t, *want.Participation, *vp.Participation, "Incorrect validator participation respond")

	url = "http://example.com?epoch=1"
	request = httptest.NewRequest(http.MethodGet, url, nil)
	writer = httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.GetValidatorParticipation(writer, request)
	assert.Equal(t, http.StatusOK, writer.Code)

	want = &structs.GetValidatorParticipationResponse{
		Participation: &structs.ValidatorParticipation{
			GlobalParticipationRate:          "1.000000",
			VotedEther:                       fmt.Sprintf("%d", validatorCount*params.BeaconConfig().MaxEffectiveBalance),
			EligibleEther:                    fmt.Sprintf("%d", validatorCount*params.BeaconConfig().MaxEffectiveBalance),
			CurrentEpochActiveGwei:           fmt.Sprintf("%d", validatorCount*params.BeaconConfig().MaxEffectiveBalance),
			CurrentEpochAttestingGwei:        fmt.Sprintf("%d", params.BeaconConfig().EffectiveBalanceIncrement), // Empty because after one epoch, current participation rotates to previous
			CurrentEpochTargetAttestingGwei:  fmt.Sprintf("%d", params.BeaconConfig().EffectiveBalanceIncrement),
			PreviousEpochActiveGwei:          fmt.Sprintf("%d", validatorCount*params.BeaconConfig().MaxEffectiveBalance),
			PreviousEpochAttestingGwei:       fmt.Sprintf("%d", validatorCount*params.BeaconConfig().MaxEffectiveBalance),
			PreviousEpochTargetAttestingGwei: fmt.Sprintf("%d", validatorCount*params.BeaconConfig().MaxEffectiveBalance),
			PreviousEpochHeadAttestingGwei:   fmt.Sprintf("%d", validatorCount*params.BeaconConfig().MaxEffectiveBalance),
		},
	}

	err = json.NewDecoder(writer.Body).Decode(&vp)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	assert.DeepEqual(t, true, vp.Finalized, "Incorrect validator participation respond")
	assert.DeepEqual(t, *want.Participation, *vp.Participation, "Incorrect validator participation respond")
}
