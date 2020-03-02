package client

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/validator/db"
	"github.com/prysmaticlabs/prysm/validator/internal"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

type mocks struct {
	validatorClient  *internal.MockBeaconNodeValidatorClient
}

func setup(t *testing.T) (*validator, *mocks, func()) {
	valDB := db.SetupDB(t, [][48]byte{validatorPubKey})
	ctrl := gomock.NewController(t)
	m := &mocks{
		validatorClient:  internal.NewMockBeaconNodeValidatorClient(ctrl),
	}
	validator := &validator{
		db:               valDB,
		validatorClient:  m.validatorClient,
		keyManager:       testKeyManager,
		graffiti:         []byte{},
		attLogs:          make(map[[32]byte]*attSubmitted),
	}

	return validator, m, ctrl.Finish
}

func TestProposeBlock_DoesNotProposeGenesisBlock(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, _, finish := setup(t)
	defer finish()
	validator.ProposeBlock(context.Background(), 0, validatorPubKey)

	testutil.AssertLogsContain(t, hook, "Assigned to genesis slot, skipping proposal")
}

func TestProposeBlock_DomainDataFailed(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(nil /*response*/, errors.New("uh oh"))

	validator.ProposeBlock(context.Background(), 1, validatorPubKey)
	testutil.AssertLogsContain(t, hook, "Failed to sign randao reveal")
}

func TestProposeBlock_RequestBlockFailed(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{}, nil /*err*/)

	m.validatorClient.EXPECT().GetBlock(
		gomock.Any(), // ctx
		gomock.Any(), // block request
	).Return(nil /*response*/, errors.New("uh oh"))

	validator.ProposeBlock(context.Background(), 1, validatorPubKey)
	testutil.AssertLogsContain(t, hook, "Failed to request block from beacon node")
}

func TestProposeBlock_ProposeBlockFailed(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Return(&ethpb.DomainResponse{}, nil /*err*/)

	m.validatorClient.EXPECT().GetBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(&ethpb.BeaconBlock{Body: &ethpb.BeaconBlockBody{}}, nil /*err*/)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Return(&ethpb.DomainResponse{}, nil /*err*/)

	m.validatorClient.EXPECT().ProposeBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.SignedBeaconBlock{}),
	).Return(nil /*response*/, errors.New("uh oh"))

	validator.ProposeBlock(context.Background(), 1, validatorPubKey)
	testutil.AssertLogsContain(t, hook, "Failed to propose block")
}

func TestProposeBlock_BlocksDoubleProposal(t *testing.T) {
	cfg := &featureconfig.Flags{
		ProtectProposer: true,
	}
	featureconfig.Init(cfg)
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()
	defer db.TeardownDB(t, validator.db)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Times(2).Return(&ethpb.DomainResponse{}, nil /*err*/)

	m.validatorClient.EXPECT().GetBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Times(2).Return(&ethpb.BeaconBlock{Body: &ethpb.BeaconBlockBody{}}, nil /*err*/)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Return(&ethpb.DomainResponse{}, nil /*err*/)

	m.validatorClient.EXPECT().ProposeBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.SignedBeaconBlock{}),
	).Return(&ethpb.ProposeResponse{}, nil /*error*/)

	validator.ProposeBlock(context.Background(), params.BeaconConfig().SlotsPerEpoch*5+2, validatorPubKey)
	testutil.AssertLogsDoNotContain(t, hook, "Tried to sign a double proposal")

	validator.ProposeBlock(context.Background(), params.BeaconConfig().SlotsPerEpoch*5+2, validatorPubKey)
	testutil.AssertLogsContain(t, hook, "Tried to sign a double proposal")
}

func TestProposeBlock_BlocksDoubleProposal_After54KEpochs(t *testing.T) {
	cfg := &featureconfig.Flags{
		ProtectProposer: true,
	}
	featureconfig.Init(cfg)
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()
	defer db.TeardownDB(t, validator.db)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Times(2).Return(&ethpb.DomainResponse{}, nil /*err*/)

	m.validatorClient.EXPECT().GetBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Times(2).Return(&ethpb.BeaconBlock{Body: &ethpb.BeaconBlockBody{}}, nil /*err*/)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Return(&ethpb.DomainResponse{}, nil /*err*/)

	m.validatorClient.EXPECT().ProposeBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.SignedBeaconBlock{}),
	).Return(&ethpb.ProposeResponse{}, nil /*error*/)

	farFuture := (params.BeaconConfig().WeakSubjectivityPeriod + 9) * params.BeaconConfig().SlotsPerEpoch
	validator.ProposeBlock(context.Background(), farFuture, validatorPubKey)
	testutil.AssertLogsDoNotContain(t, hook, "Tried to sign a double proposal")

	validator.ProposeBlock(context.Background(), farFuture, validatorPubKey)
	testutil.AssertLogsContain(t, hook, "Tried to sign a double proposal")
}

func TestProposeBlock_AllowsPastProposals(t *testing.T) {
	cfg := &featureconfig.Flags{
		ProtectProposer: true,
	}
	featureconfig.Init(cfg)
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()
	defer db.TeardownDB(t, validator.db)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Times(2).Return(&ethpb.DomainResponse{}, nil /*err*/)

	m.validatorClient.EXPECT().GetBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Times(2).Return(&ethpb.BeaconBlock{Body: &ethpb.BeaconBlockBody{}}, nil /*err*/)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Times(2).Return(&ethpb.DomainResponse{}, nil /*err*/)

	m.validatorClient.EXPECT().ProposeBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.SignedBeaconBlock{}),
	).Times(2).Return(&ethpb.ProposeResponse{}, nil /*error*/)

	farAhead := (params.BeaconConfig().WeakSubjectivityPeriod + 9) * params.BeaconConfig().SlotsPerEpoch
	validator.ProposeBlock(context.Background(), farAhead, validatorPubKey)
	testutil.AssertLogsDoNotContain(t, hook, "Tried to sign a double proposal")

	past := (params.BeaconConfig().WeakSubjectivityPeriod - 400) * params.BeaconConfig().SlotsPerEpoch
	validator.ProposeBlock(context.Background(), past, validatorPubKey)
	testutil.AssertLogsDoNotContain(t, hook, "Tried to sign a double proposal")
}

func TestProposeBlock_BroadcastsBlock(t *testing.T) {
	validator, m, finish := setup(t)
	defer finish()

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Return(&ethpb.DomainResponse{}, nil /*err*/)

	m.validatorClient.EXPECT().GetBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(&ethpb.BeaconBlock{Body: &ethpb.BeaconBlockBody{}}, nil /*err*/)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Return(&ethpb.DomainResponse{}, nil /*err*/)

	m.validatorClient.EXPECT().ProposeBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.SignedBeaconBlock{}),
	).Return(&ethpb.ProposeResponse{}, nil /*error*/)

	validator.ProposeBlock(context.Background(), 1, validatorPubKey)
}

func TestProposeBlock_BroadcastsBlock_WithGraffiti(t *testing.T) {
	validator, m, finish := setup(t)
	defer finish()

	validator.graffiti = []byte("12345678901234567890123456789012")

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Return(&ethpb.DomainResponse{}, nil /*err*/)

	m.validatorClient.EXPECT().GetBlock(
		gomock.Any(), // ctx
		gomock.Any(),
	).Return(&ethpb.BeaconBlock{Body: &ethpb.BeaconBlockBody{Graffiti: validator.graffiti}}, nil /*err*/)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), //epoch
	).Return(&ethpb.DomainResponse{}, nil /*err*/)

	var sentBlock *ethpb.SignedBeaconBlock

	m.validatorClient.EXPECT().ProposeBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&ethpb.SignedBeaconBlock{}),
	).DoAndReturn(func(ctx context.Context, block *ethpb.SignedBeaconBlock) (*ethpb.ProposeResponse, error) {
		sentBlock = block
		return &ethpb.ProposeResponse{}, nil
	})

	validator.ProposeBlock(context.Background(), 1, validatorPubKey)

	if string(sentBlock.Block.Body.Graffiti) != string(validator.graffiti) {
		t.Errorf("Block was broadcast with the wrong graffiti field, wanted \"%v\", got \"%v\"", string(validator.graffiti), string(sentBlock.Block.Body.Graffiti))
	}
}

func TestSetProposedForEpoch_SetsBit(t *testing.T) {
	wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod
	proposals := &slashpb.ProposalHistory{
		EpochBits:          bitfield.NewBitlist(wsPeriod),
		LatestEpochWritten: 0,
	}
	epoch := uint64(4)
	proposals = SetProposedForEpoch(proposals, epoch)
	proposed := HasProposedForEpoch(proposals, epoch)
	if !proposed {
		t.Fatal("Expected epoch 4 to be marked as proposed")
	}
	// Make sure no other bits are changed.
	for i := uint64(1); i <= wsPeriod; i++ {
		if i == epoch {
			continue
		}
		if HasProposedForEpoch(proposals, i) {
			t.Fatalf("Expected epoch %d to not be marked as proposed", i)
		}
	}
}

func TestSetProposedForEpoch_PrunesOverWSPeriod(t *testing.T) {
	wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod
	proposals := &slashpb.ProposalHistory{
		EpochBits:          bitfield.NewBitlist(wsPeriod),
		LatestEpochWritten: 0,
	}
	prunedEpoch := uint64(3)
	proposals = SetProposedForEpoch(proposals, prunedEpoch)

	if proposals.LatestEpochWritten != prunedEpoch {
		t.Fatalf("Expected latest epoch written to be %d, received %d", prunedEpoch, proposals.LatestEpochWritten)
	}

	epoch := wsPeriod + 4
	proposals = SetProposedForEpoch(proposals, epoch)
	if !HasProposedForEpoch(proposals, epoch) {
		t.Fatalf("Expected to be marked as proposed for epoch %d", epoch)
	}
	if proposals.LatestEpochWritten != epoch {
		t.Fatalf("Expected latest written epoch to be %d, received %d", epoch, proposals.LatestEpochWritten)
	}

	if HasProposedForEpoch(proposals, epoch-wsPeriod+prunedEpoch) {
		t.Fatalf("Expected the bit of pruned epoch %d to not be marked as proposed", epoch)
	}
	// Make sure no other bits are changed.
	for i := epoch - wsPeriod + 1; i <= epoch; i++ {
		if i == epoch {
			continue
		}
		if HasProposedForEpoch(proposals, i) {
			t.Fatalf("Expected epoch %d to not be marked as proposed", i)
		}
	}
}

func TestSetProposedForEpoch_KeepsHistory(t *testing.T) {
	wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod
	proposals := &slashpb.ProposalHistory{
		EpochBits:          bitfield.NewBitlist(wsPeriod),
		LatestEpochWritten: 0,
	}
	randomIndexes := []uint64{23, 423, 8900, 11347, 25033, 52225, 53999}
	for i := 0; i < len(randomIndexes); i++ {
		proposals = SetProposedForEpoch(proposals, randomIndexes[i])
	}
	if proposals.LatestEpochWritten != 53999 {
		t.Fatalf("Expected latest epoch written to be %d, received %d", 53999, proposals.LatestEpochWritten)
	}

	// Make sure no other bits are changed.
	for i := uint64(0); i < wsPeriod; i++ {
		setIndex := false
		for r := 0; r < len(randomIndexes); r++ {
			if i == randomIndexes[r] {
				setIndex = true
				break
			}
		}

		if setIndex != HasProposedForEpoch(proposals, i) {
			t.Fatalf("Expected epoch %d to be marked as %t", i, setIndex)
		}
	}

	// Set a past epoch as proposed, and make sure the recent data isn't changed.
	proposals = SetProposedForEpoch(proposals, randomIndexes[1]+5)
	if proposals.LatestEpochWritten != 53999 {
		t.Fatalf("Expected last epoch written to not change after writing a past epoch, received %d", proposals.LatestEpochWritten)
	}
	// Proposal just marked should be true.
	if !HasProposedForEpoch(proposals, randomIndexes[1]+5) {
		t.Fatal("Expected marked past epoch to be true, received false")
	}
	// Previously marked proposal should stay true.
	if !HasProposedForEpoch(proposals, randomIndexes[1]) {
		t.Fatal("Expected marked past epoch to be true, received false")
	}
}

func TestSetProposedForEpoch_PreventsProposingFutureEpochs(t *testing.T) {
	wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod
	proposals := &slashpb.ProposalHistory{
		EpochBits:          bitfield.NewBitlist(wsPeriod),
		LatestEpochWritten: 0,
	}
	proposals = SetProposedForEpoch(proposals, 200)
	if HasProposedForEpoch(proposals, wsPeriod+200) {
		t.Fatalf("Expected epoch %d to not be marked as proposed", wsPeriod+200)
	}
}
