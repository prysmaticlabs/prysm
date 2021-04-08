package beacon

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/subscriber/api/events"
	"strconv"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestServer_ListAssignments_CannotRequestFutureEpoch(t *testing.T) {

	db := dbTest.SetupDB(t)
	ctx := context.Background()
	bs := &Server{
		BeaconDB:           db,
		GenesisTimeFetcher: &mock.ChainService{},
	}

	wanted := errNoEpochInfoError
	_, err := bs.ListValidatorAssignments(
		ctx,
		&ethpb.ListValidatorAssignmentsRequest{
			QueryFilter: &ethpb.ListValidatorAssignmentsRequest_Epoch{
				Epoch: helpers.SlotToEpoch(bs.GenesisTimeFetcher.CurrentSlot()) + 1,
			},
		},
	)
	assert.ErrorContains(t, wanted, err)
}

func TestServer_ListAssignments_NoResults(t *testing.T) {

	db := dbTest.SetupDB(t)
	ctx := context.Background()
	st, err := testutil.NewBeaconState()
	require.NoError(t, err)

	b := testutil.NewBeaconBlock()
	require.NoError(t, db.SaveBlock(ctx, b))
	gRoot, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, gRoot))
	require.NoError(t, db.SaveState(ctx, st, gRoot))

	bs := &Server{
		BeaconDB:           db,
		GenesisTimeFetcher: &mock.ChainService{},
		StateGen:           stategen.New(db),
	}
	wanted := &ethpb.ValidatorAssignments{
		Assignments:   make([]*ethpb.ValidatorAssignments_CommitteeAssignment, 0),
		TotalSize:     int32(0),
		NextPageToken: strconv.Itoa(0),
	}
	res, err := bs.ListValidatorAssignments(
		ctx,
		&ethpb.ListValidatorAssignmentsRequest{
			QueryFilter: &ethpb.ListValidatorAssignmentsRequest_Genesis{
				Genesis: true,
			},
		},
	)
	require.NoError(t, err)
	if !proto.Equal(wanted, res) {
		t.Errorf("Wanted %v, received %v", wanted, res)
	}
}

func TestServer_ListAssignments_Pagination_InputOutOfRange(t *testing.T) {
	helpers.ClearCache()
	db := dbTest.SetupDB(t)
	ctx := context.Background()
	count := 100
	validators := make([]*ethpb.Validator, 0, count)
	for i := 0; i < count; i++ {
		pubKey := make([]byte, params.BeaconConfig().BLSPubkeyLength)
		withdrawalCred := make([]byte, 32)
		binary.LittleEndian.PutUint64(pubKey, uint64(i))
		validators = append(validators, &ethpb.Validator{
			PublicKey:             pubKey,
			WithdrawalCredentials: withdrawalCred,
			ExitEpoch:             params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance:      params.BeaconConfig().MaxEffectiveBalance,
			ActivationEpoch:       0,
		})
	}

	blk := testutil.NewBeaconBlock().Block
	blockRoot, err := blk.HashTreeRoot()
	require.NoError(t, err)

	s, err := testutil.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, s.SetValidators(validators))
	require.NoError(t, db.SaveState(ctx, s, blockRoot))
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, blockRoot))

	bs := &Server{
		BeaconDB: db,
		HeadFetcher: &mock.ChainService{
			State: s,
		},
		FinalizationFetcher: &mock.ChainService{
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			},
		},
		GenesisTimeFetcher: &mock.ChainService{},
		StateGen:           stategen.New(db),
	}

	wanted := fmt.Sprintf("page start %d >= list %d", 500, count)
	_, err = bs.ListValidatorAssignments(context.Background(), &ethpb.ListValidatorAssignmentsRequest{
		PageToken:   strconv.Itoa(2),
		QueryFilter: &ethpb.ListValidatorAssignmentsRequest_Genesis{Genesis: true},
	})
	assert.ErrorContains(t, wanted, err)
}

func TestServer_ListAssignments_Pagination_ExceedsMaxPageSize(t *testing.T) {
	bs := &Server{}
	exceedsMax := int32(cmd.Get().MaxRPCPageSize + 1)

	wanted := fmt.Sprintf("Requested page size %d can not be greater than max size %d", exceedsMax, cmd.Get().MaxRPCPageSize)
	req := &ethpb.ListValidatorAssignmentsRequest{
		PageToken: strconv.Itoa(0),
		PageSize:  exceedsMax,
	}
	_, err := bs.ListValidatorAssignments(context.Background(), req)
	assert.ErrorContains(t, wanted, err)
}

func TestServer_ListAssignments_Pagination_DefaultPageSize_NoArchive(t *testing.T) {
	helpers.ClearCache()
	db := dbTest.SetupDB(t)
	ctx := context.Background()
	count := 500
	validators := make([]*ethpb.Validator, 0, count)
	for i := 0; i < count; i++ {
		pubKey := make([]byte, params.BeaconConfig().BLSPubkeyLength)
		withdrawalCred := make([]byte, 32)
		binary.LittleEndian.PutUint64(pubKey, uint64(i))
		// Mark the validators with index divisible by 3 inactive.
		if i%3 == 0 {
			validators = append(validators, &ethpb.Validator{
				PublicKey:             pubKey,
				WithdrawalCredentials: withdrawalCred,
				ExitEpoch:             0,
				ActivationEpoch:       0,
				EffectiveBalance:      params.BeaconConfig().MaxEffectiveBalance,
			})
		} else {
			validators = append(validators, &ethpb.Validator{
				PublicKey:             pubKey,
				WithdrawalCredentials: withdrawalCred,
				ExitEpoch:             params.BeaconConfig().FarFutureEpoch,
				EffectiveBalance:      params.BeaconConfig().MaxEffectiveBalance,
				ActivationEpoch:       0,
			})
		}
	}

	blk := testutil.NewBeaconBlock().Block
	blockRoot, err := blk.HashTreeRoot()
	require.NoError(t, err)

	s, err := testutil.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, s.SetValidators(validators))
	require.NoError(t, db.SaveState(ctx, s, blockRoot))
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, blockRoot))

	bs := &Server{
		BeaconDB: db,
		HeadFetcher: &mock.ChainService{
			State: s,
		},
		FinalizationFetcher: &mock.ChainService{
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			},
		},
		GenesisTimeFetcher: &mock.ChainService{},
		StateGen:           stategen.New(db),
	}

	res, err := bs.ListValidatorAssignments(context.Background(), &ethpb.ListValidatorAssignmentsRequest{
		QueryFilter: &ethpb.ListValidatorAssignmentsRequest_Genesis{Genesis: true},
	})
	require.NoError(t, err)

	// Construct the wanted assignments.
	var wanted []*ethpb.ValidatorAssignments_CommitteeAssignment

	activeIndices, err := helpers.ActiveValidatorIndices(s, 0)
	require.NoError(t, err)
	committeeAssignments, proposerIndexToSlots, err := helpers.CommitteeAssignments(s, 0)
	require.NoError(t, err)
	for _, index := range activeIndices[0:params.BeaconConfig().DefaultPageSize] {
		val, err := s.ValidatorAtIndex(index)
		require.NoError(t, err)
		wanted = append(wanted, &ethpb.ValidatorAssignments_CommitteeAssignment{
			BeaconCommittees: committeeAssignments[index].Committee,
			CommitteeIndex:   committeeAssignments[index].CommitteeIndex,
			AttesterSlot:     committeeAssignments[index].AttesterSlot,
			ProposerSlots:    proposerIndexToSlots[index],
			PublicKey:        val.PublicKey,
			ValidatorIndex:   index,
		})
	}
	assert.DeepEqual(t, wanted, res.Assignments, "Did not receive wanted assignments")
}

func TestServer_ListAssignments_FilterPubkeysIndices_NoPagination(t *testing.T) {
	helpers.ClearCache()
	db := dbTest.SetupDB(t)

	ctx := context.Background()
	count := 100
	validators := make([]*ethpb.Validator, 0, count)
	withdrawCreds := make([]byte, 32)
	for i := 0; i < count; i++ {
		pubKey := make([]byte, params.BeaconConfig().BLSPubkeyLength)
		binary.LittleEndian.PutUint64(pubKey, uint64(i))
		val := &ethpb.Validator{
			PublicKey:             pubKey,
			WithdrawalCredentials: withdrawCreds,
			ExitEpoch:             params.BeaconConfig().FarFutureEpoch,
		}
		validators = append(validators, val)
	}

	blk := testutil.NewBeaconBlock().Block
	blockRoot, err := blk.HashTreeRoot()
	require.NoError(t, err)
	s, err := testutil.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, s.SetValidators(validators))
	require.NoError(t, db.SaveState(ctx, s, blockRoot))
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, blockRoot))

	bs := &Server{
		BeaconDB: db,
		FinalizationFetcher: &mock.ChainService{
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			},
		},
		GenesisTimeFetcher: &mock.ChainService{},
		StateGen:           stategen.New(db),
	}

	pubKey1 := make([]byte, params.BeaconConfig().BLSPubkeyLength)
	binary.LittleEndian.PutUint64(pubKey1, 1)
	pubKey2 := make([]byte, params.BeaconConfig().BLSPubkeyLength)
	binary.LittleEndian.PutUint64(pubKey2, 2)
	req := &ethpb.ListValidatorAssignmentsRequest{PublicKeys: [][]byte{pubKey1, pubKey2}, Indices: []types.ValidatorIndex{2, 3}}
	res, err := bs.ListValidatorAssignments(context.Background(), req)
	require.NoError(t, err)

	// Construct the wanted assignments.
	var wanted []*ethpb.ValidatorAssignments_CommitteeAssignment

	activeIndices, err := helpers.ActiveValidatorIndices(s, 0)
	require.NoError(t, err)
	committeeAssignments, proposerIndexToSlots, err := helpers.CommitteeAssignments(s, 0)
	require.NoError(t, err)
	for _, index := range activeIndices[1:4] {
		val, err := s.ValidatorAtIndex(index)
		require.NoError(t, err)
		wanted = append(wanted, &ethpb.ValidatorAssignments_CommitteeAssignment{
			BeaconCommittees: committeeAssignments[index].Committee,
			CommitteeIndex:   committeeAssignments[index].CommitteeIndex,
			AttesterSlot:     committeeAssignments[index].AttesterSlot,
			ProposerSlots:    proposerIndexToSlots[index],
			PublicKey:        val.PublicKey,
			ValidatorIndex:   index,
		})
	}

	assert.DeepEqual(t, wanted, res.Assignments, "Did not receive wanted assignments")
}

func TestServer_ListAssignments_CanFilterPubkeysIndices_WithPagination(t *testing.T) {
	helpers.ClearCache()
	db := dbTest.SetupDB(t)
	ctx := context.Background()
	count := 100
	validators := make([]*ethpb.Validator, 0, count)
	withdrawCred := make([]byte, 32)
	for i := 0; i < count; i++ {
		pubKey := make([]byte, params.BeaconConfig().BLSPubkeyLength)
		binary.LittleEndian.PutUint64(pubKey, uint64(i))
		val := &ethpb.Validator{
			PublicKey:             pubKey,
			WithdrawalCredentials: withdrawCred,
			ExitEpoch:             params.BeaconConfig().FarFutureEpoch,
		}
		validators = append(validators, val)
	}

	blk := testutil.NewBeaconBlock().Block
	blockRoot, err := blk.HashTreeRoot()
	require.NoError(t, err)
	s, err := testutil.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, s.SetValidators(validators))
	require.NoError(t, db.SaveState(ctx, s, blockRoot))
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, blockRoot))

	bs := &Server{
		BeaconDB: db,
		FinalizationFetcher: &mock.ChainService{
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			},
		},
		GenesisTimeFetcher: &mock.ChainService{},
		StateGen:           stategen.New(db),
	}

	req := &ethpb.ListValidatorAssignmentsRequest{Indices: []types.ValidatorIndex{1, 2, 3, 4, 5, 6}, PageSize: 2, PageToken: "1"}
	res, err := bs.ListValidatorAssignments(context.Background(), req)
	require.NoError(t, err)

	// Construct the wanted assignments.
	var assignments []*ethpb.ValidatorAssignments_CommitteeAssignment

	activeIndices, err := helpers.ActiveValidatorIndices(s, 0)
	require.NoError(t, err)
	committeeAssignments, proposerIndexToSlots, err := helpers.CommitteeAssignments(s, 0)
	require.NoError(t, err)
	for _, index := range activeIndices[3:5] {
		val, err := s.ValidatorAtIndex(index)
		require.NoError(t, err)
		assignments = append(assignments, &ethpb.ValidatorAssignments_CommitteeAssignment{
			BeaconCommittees: committeeAssignments[index].Committee,
			CommitteeIndex:   committeeAssignments[index].CommitteeIndex,
			AttesterSlot:     committeeAssignments[index].AttesterSlot,
			ProposerSlots:    proposerIndexToSlots[index],
			PublicKey:        val.PublicKey,
			ValidatorIndex:   index,
		})
	}

	wantedRes := &ethpb.ValidatorAssignments{
		Assignments:   assignments,
		TotalSize:     int32(len(req.Indices)),
		NextPageToken: "2",
	}

	assert.DeepEqual(t, wantedRes, res, "Did not get wanted assignments")

	// Test the wrap around scenario.
	assignments = nil
	req = &ethpb.ListValidatorAssignmentsRequest{Indices: []types.ValidatorIndex{1, 2, 3, 4, 5, 6}, PageSize: 5, PageToken: "1"}
	res, err = bs.ListValidatorAssignments(context.Background(), req)
	require.NoError(t, err)
	cAssignments, proposerIndexToSlots, err := helpers.CommitteeAssignments(s, 0)
	require.NoError(t, err)
	for _, index := range activeIndices[6:7] {
		val, err := s.ValidatorAtIndex(index)
		require.NoError(t, err)
		assignments = append(assignments, &ethpb.ValidatorAssignments_CommitteeAssignment{
			BeaconCommittees: cAssignments[index].Committee,
			CommitteeIndex:   cAssignments[index].CommitteeIndex,
			AttesterSlot:     cAssignments[index].AttesterSlot,
			ProposerSlots:    proposerIndexToSlots[index],
			PublicKey:        val.PublicKey,
			ValidatorIndex:   index,
		})
	}

	wantedRes = &ethpb.ValidatorAssignments{
		Assignments:   assignments,
		TotalSize:     int32(len(req.Indices)),
		NextPageToken: "",
	}

	assert.DeepEqual(t, wantedRes, res, "Did not receive wanted assignments")
}

func TestServer_NextEpochProposerList(t *testing.T) {
	helpers.ClearCache()
	db := dbTest.SetupDB(t)
	genTime := time.Now()
	config := params.BeaconConfig().Copy()
	oldConfig := config.Copy()
	config.SlotsPerEpoch = 32
	params.OverrideBeaconConfig(config)

	defer func() {
		params.OverrideBeaconConfig(oldConfig)
	}()

	bs := &Server{
		Ctx:      context.Background(),
		BeaconDB: db,
		FinalizationFetcher: &mock.ChainService{
			Genesis: genTime,
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			},
		},
		GenesisTimeFetcher: &mock.ChainService{
			Genesis: genTime,
		},
		StateGen: stategen.New(db),
	}

	count := 10000
	validators := make([]*ethpb.Validator, 0, count)
	withdrawCred := make([]byte, 32)
	for i := 0; i < count; i++ {
		pubKey := make([]byte, params.BeaconConfig().BLSPubkeyLength)
		binary.LittleEndian.PutUint64(pubKey, uint64(i))
		val := &ethpb.Validator{
			PublicKey:             pubKey,
			WithdrawalCredentials: withdrawCred,
			ExitEpoch:             params.BeaconConfig().FarFutureEpoch,
		}
		validators = append(validators, val)
	}

	blk := testutil.NewBeaconBlock().Block
	parentRoot := [32]byte{1, 2, 3}
	blk.ParentRoot = parentRoot[:]
	blockRoot, err := blk.HashTreeRoot()
	require.NoError(t, err)
	s, err := testutil.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, s.SetValidators(validators))
	require.NoError(t, db.SaveState(bs.Ctx, s, blockRoot))
	require.NoError(t, db.SaveGenesisBlockRoot(bs.Ctx, blockRoot))

	parentRoot = blockRoot
	minimalConsensusInfo, err := bs.FutureEpochProposerList(bs.Ctx)
	require.NoError(t, err)
	// We should get epoch 1
	require.Equal(t, int(config.SlotsPerEpoch), len(minimalConsensusInfo.ValidatorList))
}

func TestServer_MinimalConsensusSuite(t *testing.T) {
	helpers.ClearCache()
	db := dbTest.SetupDB(t)
	ctx := context.Background()
	count := 10000
	validators := make([]*ethpb.Validator, 0, count)
	withdrawCred := make([]byte, 32)
	for i := 0; i < count; i++ {
		pubKey := make([]byte, params.BeaconConfig().BLSPubkeyLength)
		binary.LittleEndian.PutUint64(pubKey, uint64(i))
		val := &ethpb.Validator{
			PublicKey:             pubKey,
			WithdrawalCredentials: withdrawCred,
			ExitEpoch:             params.BeaconConfig().FarFutureEpoch,
		}
		validators = append(validators, val)
	}

	config := params.BeaconConfig().Copy()
	oldConfig := config.Copy()
	config.SlotsPerEpoch = 32
	params.OverrideBeaconConfig(config)

	defer func() {
		params.OverrideBeaconConfig(oldConfig)
	}()

	testStartTime := time.Now()
	validTestEpochs := 5
	totalSec := int64(params.BeaconConfig().SlotsPerEpoch.Mul(uint64(validTestEpochs) * params.BeaconConfig().SecondsPerSlot))
	genTime := testStartTime.Unix() - totalSec
	blks := make([]*ethpb.SignedBeaconBlock, validTestEpochs*int(params.BeaconConfig().SlotsPerEpoch))
	lastValidTestSlot := params.BeaconConfig().SlotsPerEpoch * types.Slot(validTestEpochs)
	parentRoot := [32]byte{1, 2, 3}

	blk := testutil.NewBeaconBlock().Block
	blk.ParentRoot = parentRoot[:]
	blockRoot, err := blk.HashTreeRoot()
	require.NoError(t, err)
	s, err := testutil.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, s.SetValidators(validators))
	require.NoError(t, db.SaveState(ctx, s, blockRoot))
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, blockRoot))

	parentRoot = blockRoot

	bs := &Server{
		Ctx:      ctx,
		BeaconDB: db,
		FinalizationFetcher: &mock.ChainService{
			Genesis: time.Unix(genTime, 0),
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			},
		},
		GenesisTimeFetcher: &mock.ChainService{
			Genesis: time.Unix(genTime, 0),
		},
		StateGen: stategen.New(db),
	}

	for i := types.Slot(0); i < lastValidTestSlot; i++ {
		b := testutil.NewBeaconBlock()
		b.Block.Slot = i
		b.Block.ParentRoot = parentRoot[:]
		blks[i] = b
		currentRoot, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		parentRoot = currentRoot

		st, err := testutil.NewBeaconState()
		require.NoError(t, err)
		require.NoError(t, st.SetSlot(i))
		require.NoError(t, st.SetValidators(validators))
		require.NoError(t, db.SaveState(ctx, st, currentRoot))
		assert.Equal(t, true, db.HasState(ctx, currentRoot))
		hasState, err := bs.StateGen.HasState(ctx, currentRoot)
		require.NoError(t, err)
		assert.Equal(t, true, hasState)
	}

	require.NoError(t, db.SaveBlocks(ctx, blks))

	t.Run("should GetMinimalConsensusInfo", func(t *testing.T) {
		for epoch := types.Epoch(0); epoch < types.Epoch(validTestEpochs); epoch++ {
			assignments, err := bs.GetMinimalConsensusInfo(ctx, epoch)
			require.NoError(t, err)
			assert.Equal(t, epoch, types.Epoch(assignments.Epoch))

			if types.Epoch(0) == epoch {
				publicKeyBytes := make([]byte, params.BeaconConfig().BLSPubkeyLength)
				currentString := fmt.Sprintf("0x%s", hex.EncodeToString(publicKeyBytes))
				assert.Equal(t, currentString, assignments.ValidatorList[0])
			}
		}
	})

	t.Run("should not GetMinimalConsensusInfo for future epoch", func(t *testing.T) {
		epoch := types.Epoch(validTestEpochs + 1)
		minimalConsensusInfo, err := bs.GetMinimalConsensusInfo(ctx, epoch)
		assert.NotNil(t, err)
		assert.DeepEqual(t, (*events.MinimalEpochConsensusInfo)(nil), minimalConsensusInfo)
	})

	t.Run("should GetMinimalConsensusInfoRange", func(t *testing.T) {
		consensusInformation, err := bs.GetMinimalConsensusInfoRange(ctx, 0)
		require.NoError(t, err)
		require.DeepEqual(t, validTestEpochs, len(consensusInformation))
	})

	t.Run("should fail GetMinimalConsensusInfoRange", func(t *testing.T) {
		consensusInformation, err := bs.GetMinimalConsensusInfoRange(ctx, types.Epoch(validTestEpochs+1))
		require.NotNil(t, err)
		require.DeepEqual(t, 0, len(consensusInformation))
	})
}
