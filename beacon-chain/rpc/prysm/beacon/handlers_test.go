package beacon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	chainMock "github.com/prysmaticlabs/prysm/v5/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	dbTest "github.com/prysmaticlabs/prysm/v5/beacon-chain/db/testing"
	doublylinkedtree "github.com/prysmaticlabs/prysm/v5/beacon-chain/forkchoice/doubly-linked-tree"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/rpc/core"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state/stategen"
	mockstategen "github.com/prysmaticlabs/prysm/v5/beacon-chain/state/stategen/mock"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
)

func individualVotesHelper(t *testing.T, request *structs.GetIndividualVotesRequest, s *Server) (string, *structs.GetIndividualVotesResponse) {
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(request)
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(s.GetIndividualVotes))
	defer srv.Close()
	req := httptest.NewRequest(
		http.MethodGet,
		"http://example.com/eth/v1/beacon/individual_votes",
		&buf,
	)
	client := &http.Client{}
	rawResp, err := client.Post(srv.URL, "application/json", req.Body)
	require.NoError(t, err)
	defer func() {
		if err := rawResp.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	body, err := io.ReadAll(rawResp.Body)
	require.NoError(t, err)
	type ErrorResponse struct {
		Message string `json:"message"`
	}
	if rawResp.StatusCode != 200 {
		var errorResponse ErrorResponse
		err = json.Unmarshal(body, &errorResponse)
		require.NoError(t, err)
		return errorResponse.Message, &structs.GetIndividualVotesResponse{}
	}
	var votes *structs.GetIndividualVotesResponse
	err = json.Unmarshal(body, &votes)
	require.NoError(t, err)
	return "", votes
}

func TestServer_GetIndividualVotes_RequestFutureSlot(t *testing.T) {
	s := &Server{
		CoreService: &core.Service{
			GenesisTimeFetcher: &chainMock.ChainService{},
		},
	}
	request := &structs.GetIndividualVotesRequest{
		Epoch: fmt.Sprintf("%d", slots.ToEpoch(s.CoreService.GenesisTimeFetcher.CurrentSlot())+1),
	}
	errorResp, _ := individualVotesHelper(t, request, s)
	require.StringContains(t, "cannot retrieve information about an epoch in the future", errorResp)
}

func addDefaultReplayerBuilder(s *Server, h stategen.HistoryAccessor) {
	cc := &mockstategen.CanonicalChecker{Is: true, Err: nil}
	cs := &mockstategen.CurrentSlotter{Slot: math.MaxUint64 - 1}
	s.CoreService.ReplayerBuilder = stategen.NewCanonicalHistory(h, cc, cs)
}

func TestServer_GetIndividualVotes_ValidatorsDontExist(t *testing.T) {
	beaconDB := dbTest.SetupDB(t)
	ctx := context.Background()

	var slot primitives.Slot = 0
	validators := uint64(64)
	stateWithValidators, _ := util.DeterministicGenesisState(t, validators)
	beaconState, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, beaconState.SetValidators(stateWithValidators.Validators()))
	require.NoError(t, beaconState.SetSlot(slot))

	b := util.NewBeaconBlock()
	b.Block.Slot = slot
	util.SaveBlock(t, ctx, beaconDB, b)
	gRoot, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	gen := stategen.New(beaconDB, doublylinkedtree.New())
	require.NoError(t, gen.SaveState(ctx, gRoot, beaconState))
	require.NoError(t, beaconDB.SaveState(ctx, beaconState, gRoot))
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, gRoot))
	s := &Server{
		CoreService: &core.Service{
			StateGen:           gen,
			GenesisTimeFetcher: &chainMock.ChainService{},
		},
	}
	addDefaultReplayerBuilder(s, beaconDB)

	// Test non exist public key.
	request := &structs.GetIndividualVotesRequest{
		PublicKeys: []string{"0xaa"},
		Epoch:      "0",
	}
	errStr, resp := individualVotesHelper(t, request, s)
	require.Equal(t, "", errStr)
	want := &structs.GetIndividualVotesResponse{
		IndividualVotes: []*structs.IndividualVote{
			{
				Epoch:                            "0",
				PublicKey:                        "0xaa",
				ValidatorIndex:                   fmt.Sprintf("%d", ^uint64(0)),
				CurrentEpochEffectiveBalanceGwei: "0",
				InclusionSlot:                    "0",
				InclusionDistance:                "0",
				InactivityScore:                  "0",
			},
		},
	}
	assert.DeepEqual(t, want, resp, "Unexpected response")

	// Test non-existent validator index.
	request = &structs.GetIndividualVotesRequest{
		Indices: []string{"100"},
		Epoch:   "0",
	}
	errStr, resp = individualVotesHelper(t, request, s)
	require.Equal(t, "", errStr)
	want = &structs.GetIndividualVotesResponse{
		IndividualVotes: []*structs.IndividualVote{
			{
				Epoch:                            "0",
				PublicKey:                        "0x",
				ValidatorIndex:                   "100",
				CurrentEpochEffectiveBalanceGwei: "0",
				InclusionSlot:                    "0",
				InclusionDistance:                "0",
				InactivityScore:                  "0",
			},
		},
	}
	assert.DeepEqual(t, want, resp, "Unexpected response")

	// Test both.
	request = &structs.GetIndividualVotesRequest{
		PublicKeys: []string{"0xaa", "0xbb"},
		Indices:    []string{"100", "101"},
		Epoch:      "0",
	}
	errStr, resp = individualVotesHelper(t, request, s)
	require.Equal(t, "", errStr)
	want = &structs.GetIndividualVotesResponse{
		IndividualVotes: []*structs.IndividualVote{
			{Epoch: "0", PublicKey: "0xaa", ValidatorIndex: fmt.Sprintf("%d", ^uint64(0)), CurrentEpochEffectiveBalanceGwei: "0", InclusionSlot: "0", InclusionDistance: "0", InactivityScore: "0"},
			{
				Epoch:                            "0",
				PublicKey:                        "0xbb",
				ValidatorIndex:                   fmt.Sprintf("%d", ^uint64(0)),
				CurrentEpochEffectiveBalanceGwei: "0",
				InclusionSlot:                    "0",
				InclusionDistance:                "0",
				InactivityScore:                  "0",
			},
			{
				Epoch:                            "0",
				PublicKey:                        "0x",
				ValidatorIndex:                   "100",
				CurrentEpochEffectiveBalanceGwei: "0",
				InclusionSlot:                    "0",
				InclusionDistance:                "0",
				InactivityScore:                  "0",
			},
			{
				Epoch:                            "0",
				PublicKey:                        "0x",
				ValidatorIndex:                   "101",
				CurrentEpochEffectiveBalanceGwei: "0",
				InclusionSlot:                    "0",
				InclusionDistance:                "0",
				InactivityScore:                  "0",
			},
		},
	}
	assert.DeepEqual(t, want, resp, "Unexpected response")
}

func TestServer_GetIndividualVotes_Working(t *testing.T) {
	helpers.ClearCache()
	beaconDB := dbTest.SetupDB(t)
	ctx := context.Background()

	validators := uint64(32)
	stateWithValidators, _ := util.DeterministicGenesisState(t, validators)
	beaconState, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, beaconState.SetValidators(stateWithValidators.Validators()))

	bf := bitfield.NewBitlist(validators / uint64(params.BeaconConfig().SlotsPerEpoch))
	att1 := util.NewAttestation()
	att1.AggregationBits = bf
	att2 := util.NewAttestation()
	att2.AggregationBits = bf
	rt := [32]byte{'A'}
	att1.Data.Target.Root = rt[:]
	att1.Data.BeaconBlockRoot = rt[:]
	br := beaconState.BlockRoots()
	newRt := [32]byte{'B'}
	br[0] = newRt[:]
	require.NoError(t, beaconState.SetBlockRoots(br))
	att2.Data.Target.Root = rt[:]
	att2.Data.BeaconBlockRoot = newRt[:]
	err = beaconState.AppendPreviousEpochAttestations(&eth.PendingAttestation{
		Data: att1.Data, AggregationBits: bf, InclusionDelay: 1,
	})
	require.NoError(t, err)
	err = beaconState.AppendCurrentEpochAttestations(&eth.PendingAttestation{
		Data: att2.Data, AggregationBits: bf, InclusionDelay: 1,
	})
	require.NoError(t, err)

	b := util.NewBeaconBlock()
	b.Block.Slot = 0
	util.SaveBlock(t, ctx, beaconDB, b)
	gRoot, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	gen := stategen.New(beaconDB, doublylinkedtree.New())
	require.NoError(t, gen.SaveState(ctx, gRoot, beaconState))
	require.NoError(t, beaconDB.SaveState(ctx, beaconState, gRoot))
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, gRoot))
	s := &Server{
		CoreService: &core.Service{
			StateGen:           gen,
			GenesisTimeFetcher: &chainMock.ChainService{},
		},
	}
	addDefaultReplayerBuilder(s, beaconDB)

	request := &structs.GetIndividualVotesRequest{
		Indices: []string{"0", "1"},
		Epoch:   "0",
	}
	errStr, resp := individualVotesHelper(t, request, s)
	require.Equal(t, "", errStr)
	want := &structs.GetIndividualVotesResponse{
		IndividualVotes: []*structs.IndividualVote{
			{
				Epoch:                            "0",
				ValidatorIndex:                   "0",
				PublicKey:                        hexutil.Encode(beaconState.Validators()[0].PublicKey),
				IsActiveInCurrentEpoch:           true,
				IsActiveInPreviousEpoch:          true,
				CurrentEpochEffectiveBalanceGwei: fmt.Sprintf("%d", params.BeaconConfig().MaxEffectiveBalance),
				InclusionSlot:                    fmt.Sprintf("%d", params.BeaconConfig().FarFutureSlot),
				InclusionDistance:                fmt.Sprintf("%d", params.BeaconConfig().FarFutureSlot),
				InactivityScore:                  "0",
			},
			{
				Epoch:                            "0",
				ValidatorIndex:                   "1",
				PublicKey:                        hexutil.Encode(beaconState.Validators()[1].PublicKey),
				IsActiveInCurrentEpoch:           true,
				IsActiveInPreviousEpoch:          true,
				CurrentEpochEffectiveBalanceGwei: fmt.Sprintf("%d", params.BeaconConfig().MaxEffectiveBalance),
				InclusionSlot:                    fmt.Sprintf("%d", params.BeaconConfig().FarFutureSlot),
				InclusionDistance:                fmt.Sprintf("%d", params.BeaconConfig().FarFutureSlot),
				InactivityScore:                  "0",
			},
		},
	}
	assert.DeepEqual(t, want, resp, "Unexpected response")
}

func TestServer_GetIndividualVotes_WorkingAltair(t *testing.T) {
	helpers.ClearCache()
	beaconDB := dbTest.SetupDB(t)
	ctx := context.Background()

	var slot primitives.Slot = 0
	validators := uint64(32)
	beaconState, _ := util.DeterministicGenesisStateAltair(t, validators)
	require.NoError(t, beaconState.SetSlot(slot))

	pb, err := beaconState.CurrentEpochParticipation()
	require.NoError(t, err)
	for i := range pb {
		pb[i] = 0xff
	}
	require.NoError(t, beaconState.SetCurrentParticipationBits(pb))
	require.NoError(t, beaconState.SetPreviousParticipationBits(pb))

	b := util.NewBeaconBlock()
	b.Block.Slot = slot
	util.SaveBlock(t, ctx, beaconDB, b)
	gRoot, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	gen := stategen.New(beaconDB, doublylinkedtree.New())
	require.NoError(t, gen.SaveState(ctx, gRoot, beaconState))
	require.NoError(t, beaconDB.SaveState(ctx, beaconState, gRoot))
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, gRoot))
	s := &Server{
		CoreService: &core.Service{
			StateGen:           gen,
			GenesisTimeFetcher: &chainMock.ChainService{},
		},
	}
	addDefaultReplayerBuilder(s, beaconDB)

	request := &structs.GetIndividualVotesRequest{
		Indices: []string{"0", "1"},
		Epoch:   "0",
	}
	errStr, resp := individualVotesHelper(t, request, s)
	require.Equal(t, "", errStr)
	want := &structs.GetIndividualVotesResponse{
		IndividualVotes: []*structs.IndividualVote{
			{
				Epoch:                            "0",
				ValidatorIndex:                   "0",
				PublicKey:                        hexutil.Encode(beaconState.Validators()[0].PublicKey),
				IsActiveInCurrentEpoch:           true,
				IsActiveInPreviousEpoch:          true,
				IsCurrentEpochTargetAttester:     true,
				IsCurrentEpochAttester:           true,
				IsPreviousEpochAttester:          true,
				IsPreviousEpochHeadAttester:      true,
				IsPreviousEpochTargetAttester:    true,
				CurrentEpochEffectiveBalanceGwei: fmt.Sprintf("%d", params.BeaconConfig().MaxEffectiveBalance),
				InclusionSlot:                    "0",
				InclusionDistance:                "0",
				InactivityScore:                  "0",
			},
			{
				Epoch:                            "0",
				ValidatorIndex:                   "1",
				PublicKey:                        hexutil.Encode(beaconState.Validators()[1].PublicKey),
				IsActiveInCurrentEpoch:           true,
				IsActiveInPreviousEpoch:          true,
				IsCurrentEpochTargetAttester:     true,
				IsCurrentEpochAttester:           true,
				IsPreviousEpochAttester:          true,
				IsPreviousEpochHeadAttester:      true,
				IsPreviousEpochTargetAttester:    true,
				CurrentEpochEffectiveBalanceGwei: fmt.Sprintf("%d", params.BeaconConfig().MaxEffectiveBalance),
				InclusionSlot:                    "0",
				InclusionDistance:                "0",
				InactivityScore:                  "0",
			},
		},
	}
	assert.DeepEqual(t, want, resp, "Unexpected response")
}

func TestServer_GetIndividualVotes_AltairEndOfEpoch(t *testing.T) {
	helpers.ClearCache()
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.BeaconConfig())
	beaconDB := dbTest.SetupDB(t)
	ctx := context.Background()

	validators := uint64(32)
	beaconState, _ := util.DeterministicGenesisStateAltair(t, validators)
	startSlot, err := slots.EpochStart(1)
	assert.NoError(t, err)
	require.NoError(t, beaconState.SetSlot(startSlot))

	b := util.NewBeaconBlock()
	b.Block.Slot = startSlot
	util.SaveBlock(t, ctx, beaconDB, b)
	gRoot, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	gen := stategen.New(beaconDB, doublylinkedtree.New())
	require.NoError(t, gen.SaveState(ctx, gRoot, beaconState))
	require.NoError(t, beaconDB.SaveState(ctx, beaconState, gRoot))
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, gRoot))
	// Save State at the end of the epoch:
	endSlot, err := slots.EpochEnd(1)
	assert.NoError(t, err)

	beaconState, _ = util.DeterministicGenesisStateAltair(t, validators)
	require.NoError(t, beaconState.SetSlot(endSlot))

	pb, err := beaconState.CurrentEpochParticipation()
	require.NoError(t, err)
	for i := range pb {
		pb[i] = 0xff
	}
	require.NoError(t, beaconState.SetCurrentParticipationBits(pb))
	require.NoError(t, beaconState.SetPreviousParticipationBits(pb))

	b.Block.Slot = endSlot
	util.SaveBlock(t, ctx, beaconDB, b)
	gRoot, err = b.Block.HashTreeRoot()
	require.NoError(t, err)

	require.NoError(t, gen.SaveState(ctx, gRoot, beaconState))
	require.NoError(t, beaconDB.SaveState(ctx, beaconState, gRoot))
	s := &Server{
		CoreService: &core.Service{
			StateGen:           gen,
			GenesisTimeFetcher: &chainMock.ChainService{},
		},
	}
	addDefaultReplayerBuilder(s, beaconDB)

	request := &structs.GetIndividualVotesRequest{
		Indices: []string{"0", "1"},
		Epoch:   "1",
	}
	errStr, resp := individualVotesHelper(t, request, s)
	require.Equal(t, "", errStr)
	want := &structs.GetIndividualVotesResponse{
		IndividualVotes: []*structs.IndividualVote{
			{
				Epoch:                            "1",
				ValidatorIndex:                   "0",
				PublicKey:                        hexutil.Encode(beaconState.Validators()[0].PublicKey),
				IsActiveInCurrentEpoch:           true,
				IsActiveInPreviousEpoch:          true,
				IsCurrentEpochTargetAttester:     true,
				IsCurrentEpochAttester:           true,
				IsPreviousEpochAttester:          true,
				IsPreviousEpochHeadAttester:      true,
				IsPreviousEpochTargetAttester:    true,
				CurrentEpochEffectiveBalanceGwei: fmt.Sprintf("%d", params.BeaconConfig().MaxEffectiveBalance),
				InclusionSlot:                    "0",
				InclusionDistance:                "0",
				InactivityScore:                  "0",
			},
			{
				Epoch:                            "1",
				ValidatorIndex:                   "1",
				PublicKey:                        hexutil.Encode(beaconState.Validators()[1].PublicKey),
				IsActiveInCurrentEpoch:           true,
				IsActiveInPreviousEpoch:          true,
				IsCurrentEpochTargetAttester:     true,
				IsCurrentEpochAttester:           true,
				IsPreviousEpochAttester:          true,
				IsPreviousEpochHeadAttester:      true,
				IsPreviousEpochTargetAttester:    true,
				CurrentEpochEffectiveBalanceGwei: fmt.Sprintf("%d", params.BeaconConfig().MaxEffectiveBalance),
				InclusionSlot:                    "0",
				InclusionDistance:                "0",
				InactivityScore:                  "0",
			},
		},
	}
	assert.DeepEqual(t, want, resp, "Unexpected response")
}

func TestServer_GetIndividualVotes_BellatrixEndOfEpoch(t *testing.T) {
	helpers.ClearCache()
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.BeaconConfig())
	beaconDB := dbTest.SetupDB(t)
	ctx := context.Background()

	validators := uint64(32)
	beaconState, _ := util.DeterministicGenesisStateBellatrix(t, validators)
	startSlot, err := slots.EpochStart(1)
	assert.NoError(t, err)
	require.NoError(t, beaconState.SetSlot(startSlot))

	b := util.NewBeaconBlock()
	b.Block.Slot = startSlot
	util.SaveBlock(t, ctx, beaconDB, b)
	gRoot, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	gen := stategen.New(beaconDB, doublylinkedtree.New())
	require.NoError(t, gen.SaveState(ctx, gRoot, beaconState))
	require.NoError(t, beaconDB.SaveState(ctx, beaconState, gRoot))
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, gRoot))
	// Save State at the end of the epoch:
	endSlot, err := slots.EpochEnd(1)
	assert.NoError(t, err)

	beaconState, _ = util.DeterministicGenesisStateBellatrix(t, validators)
	require.NoError(t, beaconState.SetSlot(endSlot))

	pb, err := beaconState.CurrentEpochParticipation()
	require.NoError(t, err)
	for i := range pb {
		pb[i] = 0xff
	}
	require.NoError(t, beaconState.SetCurrentParticipationBits(pb))
	require.NoError(t, beaconState.SetPreviousParticipationBits(pb))

	b.Block.Slot = endSlot
	util.SaveBlock(t, ctx, beaconDB, b)
	gRoot, err = b.Block.HashTreeRoot()
	require.NoError(t, err)

	require.NoError(t, gen.SaveState(ctx, gRoot, beaconState))
	require.NoError(t, beaconDB.SaveState(ctx, beaconState, gRoot))
	s := &Server{
		CoreService: &core.Service{
			StateGen:           gen,
			GenesisTimeFetcher: &chainMock.ChainService{},
		},
	}
	addDefaultReplayerBuilder(s, beaconDB)

	request := &structs.GetIndividualVotesRequest{
		Indices: []string{"0", "1"},
		Epoch:   "1",
	}
	errStr, resp := individualVotesHelper(t, request, s)
	require.Equal(t, "", errStr)
	want := &structs.GetIndividualVotesResponse{
		IndividualVotes: []*structs.IndividualVote{
			{
				Epoch:                            "1",
				ValidatorIndex:                   "0",
				PublicKey:                        hexutil.Encode(beaconState.Validators()[0].PublicKey),
				IsActiveInCurrentEpoch:           true,
				IsActiveInPreviousEpoch:          true,
				IsCurrentEpochTargetAttester:     true,
				IsCurrentEpochAttester:           true,
				IsPreviousEpochAttester:          true,
				IsPreviousEpochHeadAttester:      true,
				IsPreviousEpochTargetAttester:    true,
				CurrentEpochEffectiveBalanceGwei: fmt.Sprintf("%d", params.BeaconConfig().MaxEffectiveBalance),
				InclusionSlot:                    "0",
				InclusionDistance:                "0",
				InactivityScore:                  "0",
			},
			{
				Epoch:                            "1",
				ValidatorIndex:                   "1",
				PublicKey:                        hexutil.Encode(beaconState.Validators()[1].PublicKey),
				IsActiveInCurrentEpoch:           true,
				IsActiveInPreviousEpoch:          true,
				IsCurrentEpochTargetAttester:     true,
				IsCurrentEpochAttester:           true,
				IsPreviousEpochAttester:          true,
				IsPreviousEpochHeadAttester:      true,
				IsPreviousEpochTargetAttester:    true,
				CurrentEpochEffectiveBalanceGwei: fmt.Sprintf("%d", params.BeaconConfig().MaxEffectiveBalance),
				InclusionSlot:                    "0",
				InclusionDistance:                "0",
				InactivityScore:                  "0",
			},
		},
	}
	assert.DeepEqual(t, want, resp, "Unexpected response")
}

func TestServer_GetIndividualVotes_CapellaEndOfEpoch(t *testing.T) {
	helpers.ClearCache()
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.BeaconConfig())
	beaconDB := dbTest.SetupDB(t)
	ctx := context.Background()

	validators := uint64(32)
	beaconState, _ := util.DeterministicGenesisStateCapella(t, validators)
	startSlot, err := slots.EpochStart(1)
	assert.NoError(t, err)
	require.NoError(t, beaconState.SetSlot(startSlot))

	b := util.NewBeaconBlock()
	b.Block.Slot = startSlot
	util.SaveBlock(t, ctx, beaconDB, b)
	gRoot, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	gen := stategen.New(beaconDB, doublylinkedtree.New())
	require.NoError(t, gen.SaveState(ctx, gRoot, beaconState))
	require.NoError(t, beaconDB.SaveState(ctx, beaconState, gRoot))
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, gRoot))
	// Save State at the end of the epoch:
	endSlot, err := slots.EpochEnd(1)
	assert.NoError(t, err)

	beaconState, _ = util.DeterministicGenesisStateCapella(t, validators)
	require.NoError(t, beaconState.SetSlot(endSlot))

	pb, err := beaconState.CurrentEpochParticipation()
	require.NoError(t, err)
	for i := range pb {
		pb[i] = 0xff
	}
	require.NoError(t, beaconState.SetCurrentParticipationBits(pb))
	require.NoError(t, beaconState.SetPreviousParticipationBits(pb))

	b.Block.Slot = endSlot
	util.SaveBlock(t, ctx, beaconDB, b)
	gRoot, err = b.Block.HashTreeRoot()
	require.NoError(t, err)

	require.NoError(t, gen.SaveState(ctx, gRoot, beaconState))
	require.NoError(t, beaconDB.SaveState(ctx, beaconState, gRoot))
	s := &Server{
		CoreService: &core.Service{
			StateGen:           gen,
			GenesisTimeFetcher: &chainMock.ChainService{},
		},
	}
	addDefaultReplayerBuilder(s, beaconDB)

	request := &structs.GetIndividualVotesRequest{
		Indices: []string{"0", "1"},
		Epoch:   "1",
	}
	errStr, resp := individualVotesHelper(t, request, s)
	require.Equal(t, "", errStr)
	want := &structs.GetIndividualVotesResponse{
		IndividualVotes: []*structs.IndividualVote{
			{
				Epoch:                            "1",
				ValidatorIndex:                   "0",
				PublicKey:                        hexutil.Encode(beaconState.Validators()[0].PublicKey),
				IsActiveInCurrentEpoch:           true,
				IsActiveInPreviousEpoch:          true,
				IsCurrentEpochTargetAttester:     true,
				IsCurrentEpochAttester:           true,
				IsPreviousEpochAttester:          true,
				IsPreviousEpochHeadAttester:      true,
				IsPreviousEpochTargetAttester:    true,
				CurrentEpochEffectiveBalanceGwei: fmt.Sprintf("%d", params.BeaconConfig().MaxEffectiveBalance),
				InclusionSlot:                    "0",
				InclusionDistance:                "0",
				InactivityScore:                  "0",
			},
			{
				Epoch:                            "1",
				ValidatorIndex:                   "1",
				PublicKey:                        hexutil.Encode(beaconState.Validators()[1].PublicKey),
				IsActiveInCurrentEpoch:           true,
				IsActiveInPreviousEpoch:          true,
				IsCurrentEpochTargetAttester:     true,
				IsCurrentEpochAttester:           true,
				IsPreviousEpochAttester:          true,
				IsPreviousEpochHeadAttester:      true,
				IsPreviousEpochTargetAttester:    true,
				CurrentEpochEffectiveBalanceGwei: fmt.Sprintf("%d", params.BeaconConfig().MaxEffectiveBalance),
				InclusionSlot:                    "0",
				InclusionDistance:                "0",
				InactivityScore:                  "0",
			},
		},
	}
	assert.DeepEqual(t, want, resp, "Unexpected response")
}
