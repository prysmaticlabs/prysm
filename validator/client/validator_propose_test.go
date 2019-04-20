package client

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"testing"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/validator/internal"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

type mocks struct {
	proposerClient  *internal.MockProposerServiceClient
	beaconClient    *internal.MockBeaconServiceClient
	validatorClient *internal.MockValidatorServiceClient
	attesterClient  *internal.MockAttesterServiceClient
}

func setup(t *testing.T) (*validator, *mocks, func()) {
	ctrl := gomock.NewController(t)
	m := &mocks{
		proposerClient:  internal.NewMockProposerServiceClient(ctrl),
		beaconClient:    internal.NewMockBeaconServiceClient(ctrl),
		validatorClient: internal.NewMockValidatorServiceClient(ctrl),
		attesterClient:  internal.NewMockAttesterServiceClient(ctrl),
	}
	validator := &validator{
		proposerClient:  m.proposerClient,
		beaconClient:    m.beaconClient,
		attesterClient:  m.attesterClient,
		validatorClient: m.validatorClient,
		keys:            keyMap,
	}

	return validator, m, ctrl.Finish
}

func TestProposeBlock_DoesNotProposeGenesisBlock(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, _, finish := setup(t)
	defer finish()
	validator.ProposeBlock(context.Background(), params.BeaconConfig().GenesisSlot, hex.EncodeToString(validatorKey.PublicKey.Marshal()))

	testutil.AssertLogsContain(t, hook, "Assigned to genesis slot, skipping proposal")
}

func TestProposeBlock_LogsCanonicalHeadFailure(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()

	m.beaconClient.EXPECT().CanonicalHead(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(nil /*beaconBlock*/, errors.New("something bad happened"))

	validator.ProposeBlock(context.Background(), 55, hex.EncodeToString(validatorKey.PublicKey.Marshal()))

	testutil.AssertLogsContain(t, hook, "something bad happened")
}

func TestProposeBlock_PendingDepositsFailure(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()

	m.beaconClient.EXPECT().CanonicalHead(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pbp2p.BeaconBlock{}, nil /*err*/)

	m.beaconClient.EXPECT().PendingDeposits(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(nil /*response*/, errors.New("something bad happened"))

	validator.ProposeBlock(context.Background(), 55, hex.EncodeToString(validatorKey.PublicKey.Marshal()))

	testutil.AssertLogsContain(t, hook, "something bad happened")
}

func TestProposeBlock_UsePendingDeposits(t *testing.T) {
	validator, m, finish := setup(t)
	defer finish()

	m.beaconClient.EXPECT().CanonicalHead(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pbp2p.BeaconBlock{}, nil /*err*/)

	m.beaconClient.EXPECT().PendingDeposits(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pb.PendingDepositsResponse{
		PendingDeposits: []*pbp2p.Deposit{
			{DepositData: []byte{'D', 'A', 'T', 'A'}},
		},
	}, nil /*err*/)

	m.beaconClient.EXPECT().Eth1Data(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pb.Eth1DataResponse{}, nil /*err*/)

	m.beaconClient.EXPECT().ForkData(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pbp2p.Fork{
		Epoch:           params.BeaconConfig().GenesisEpoch,
		CurrentVersion:  0,
		PreviousVersion: 0,
	}, nil /*err*/)

	m.proposerClient.EXPECT().PendingAttestations(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pb.PendingAttestationsRequest{}),
	).Return(&pb.PendingAttestationsResponse{PendingAttestations: []*pbp2p.Attestation{}}, nil)

	m.proposerClient.EXPECT().ComputeStateRoot(
		gomock.Any(), // context
		gomock.AssignableToTypeOf(&pbp2p.BeaconBlock{}),
	).Return(&pb.StateRootResponse{
		StateRoot: []byte{'F'},
	}, nil /*err*/)

	var broadcastedBlock *pbp2p.BeaconBlock
	m.proposerClient.EXPECT().ProposeBlock(
		gomock.Any(), // context
		gomock.AssignableToTypeOf(&pbp2p.BeaconBlock{}),
	).Do(func(_ context.Context, blk *pbp2p.BeaconBlock) {
		broadcastedBlock = blk
	}).Return(&pb.ProposeResponse{}, nil /*error*/)

	validator.ProposeBlock(context.Background(), 55, hex.EncodeToString(validatorKey.PublicKey.Marshal()))

	if !bytes.Equal(broadcastedBlock.Body.Deposits[0].DepositData, []byte{'D', 'A', 'T', 'A'}) {
		t.Errorf("Unexpected deposit data: %v", broadcastedBlock.Body.Deposits)
	}
}

func TestProposeBlock_Eth1DataFailure(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()

	m.beaconClient.EXPECT().CanonicalHead(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pbp2p.BeaconBlock{}, nil /*err*/)

	m.beaconClient.EXPECT().PendingDeposits(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pb.PendingDepositsResponse{}, nil /*err*/)

	m.beaconClient.EXPECT().Eth1Data(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(nil /*response*/, errors.New("something bad happened"))

	validator.ProposeBlock(context.Background(), 55, hex.EncodeToString(validatorKey.PublicKey.Marshal()))

	testutil.AssertLogsContain(t, hook, "something bad happened")
}

func TestProposeBlock_UsesEth1Data(t *testing.T) {
	validator, m, finish := setup(t)
	defer finish()

	m.beaconClient.EXPECT().CanonicalHead(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pbp2p.BeaconBlock{}, nil /*err*/)

	m.beaconClient.EXPECT().PendingDeposits(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pb.PendingDepositsResponse{}, nil /*err*/)

	m.beaconClient.EXPECT().Eth1Data(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pb.Eth1DataResponse{
		Eth1Data: &pbp2p.Eth1Data{BlockHash32: []byte{'B', 'L', 'O', 'C', 'K'}},
	}, nil /*err*/)

	m.beaconClient.EXPECT().ForkData(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pbp2p.Fork{
		Epoch:           params.BeaconConfig().GenesisEpoch,
		CurrentVersion:  0,
		PreviousVersion: 0,
	}, nil /*err*/)

	m.proposerClient.EXPECT().PendingAttestations(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pb.PendingAttestationsRequest{}),
	).Return(&pb.PendingAttestationsResponse{PendingAttestations: []*pbp2p.Attestation{}}, nil)

	m.proposerClient.EXPECT().ComputeStateRoot(
		gomock.Any(), // context
		gomock.AssignableToTypeOf(&pbp2p.BeaconBlock{}),
	).Return(&pb.StateRootResponse{
		StateRoot: []byte{'F'},
	}, nil /*err*/)

	var broadcastedBlock *pbp2p.BeaconBlock
	m.proposerClient.EXPECT().ProposeBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pbp2p.BeaconBlock{}),
	).Do(func(_ context.Context, blk *pbp2p.BeaconBlock) {
		broadcastedBlock = blk
	}).Return(&pb.ProposeResponse{}, nil /*error*/)

	validator.ProposeBlock(context.Background(), 55, hex.EncodeToString(validatorKey.PublicKey.Marshal()))

	if !bytes.Equal(broadcastedBlock.Eth1Data.BlockHash32, []byte{'B', 'L', 'O', 'C', 'K'}) {
		t.Errorf("Unexpected ETH1 data: %v", broadcastedBlock.Eth1Data)
	}
}

func TestProposeBlock_PendingAttestations_UsesCurrentSlot(t *testing.T) {
	validator, m, finish := setup(t)
	defer finish()

	m.beaconClient.EXPECT().CanonicalHead(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pbp2p.BeaconBlock{}, nil /*err*/)

	m.beaconClient.EXPECT().PendingDeposits(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pb.PendingDepositsResponse{}, nil /*err*/)

	m.beaconClient.EXPECT().Eth1Data(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pb.Eth1DataResponse{
		Eth1Data: &pbp2p.Eth1Data{BlockHash32: []byte{'B', 'L', 'O', 'C', 'K'}},
	}, nil /*err*/)

	m.beaconClient.EXPECT().ForkData(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pbp2p.Fork{
		Epoch:           params.BeaconConfig().GenesisEpoch,
		CurrentVersion:  0,
		PreviousVersion: 0,
	}, nil /*err*/)

	var req *pb.PendingAttestationsRequest
	m.proposerClient.EXPECT().PendingAttestations(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pb.PendingAttestationsRequest{}),
	).DoAndReturn(func(_ context.Context, r *pb.PendingAttestationsRequest) (*pb.PendingAttestationsResponse, error) {
		req = r
		return &pb.PendingAttestationsResponse{PendingAttestations: []*pbp2p.Attestation{}}, nil
	})

	m.proposerClient.EXPECT().ComputeStateRoot(
		gomock.Any(), // context
		gomock.AssignableToTypeOf(&pbp2p.BeaconBlock{}),
	).Return(&pb.StateRootResponse{
		StateRoot: []byte{'F'},
	}, nil /*err*/)

	m.proposerClient.EXPECT().ProposeBlock(
		gomock.Any(), // context
		gomock.AssignableToTypeOf(&pbp2p.BeaconBlock{}),
	).Return(&pb.ProposeResponse{}, nil /*error*/)

	validator.ProposeBlock(context.Background(), 55, hex.EncodeToString(validatorKey.PublicKey.Marshal()))
	if req.ProposalBlockSlot != 55 {
		t.Errorf(
			"expected request to use the current proposal slot %d, but got %d",
			55,
			req.ProposalBlockSlot,
		)
	}
}

func TestProposeBlock_PendingAttestationsFailure(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()

	m.beaconClient.EXPECT().CanonicalHead(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pbp2p.BeaconBlock{}, nil /*err*/)

	m.beaconClient.EXPECT().PendingDeposits(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pb.PendingDepositsResponse{}, nil /*err*/)

	m.beaconClient.EXPECT().Eth1Data(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pb.Eth1DataResponse{
		Eth1Data: &pbp2p.Eth1Data{BlockHash32: []byte{'B', 'L', 'O', 'C', 'K'}},
	}, nil /*err*/)

	m.beaconClient.EXPECT().ForkData(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pbp2p.Fork{
		Epoch:           params.BeaconConfig().GenesisEpoch,
		CurrentVersion:  0,
		PreviousVersion: 0,
	}, nil /*err*/)

	m.proposerClient.EXPECT().PendingAttestations(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pb.PendingAttestationsRequest{}),
	).Return(nil, errors.New("failed"))

	validator.ProposeBlock(context.Background(), 55, hex.EncodeToString(validatorKey.PublicKey.Marshal()))
	testutil.AssertLogsContain(t, hook, "Failed to fetch pending attestations")
}

func TestProposeBlock_ComputeStateFailure(t *testing.T) {
	hook := logTest.NewGlobal()
	validator, m, finish := setup(t)
	defer finish()

	m.beaconClient.EXPECT().CanonicalHead(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pbp2p.BeaconBlock{}, nil /*err*/)

	m.beaconClient.EXPECT().PendingDeposits(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pb.PendingDepositsResponse{}, nil /*err*/)

	m.beaconClient.EXPECT().Eth1Data(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pb.Eth1DataResponse{}, nil /*err*/)

	m.beaconClient.EXPECT().ForkData(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pbp2p.Fork{
		Epoch:           params.BeaconConfig().GenesisEpoch,
		CurrentVersion:  0,
		PreviousVersion: 0,
	}, nil /*err*/)

	m.proposerClient.EXPECT().PendingAttestations(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pb.PendingAttestationsRequest{}),
	).Return(&pb.PendingAttestationsResponse{PendingAttestations: []*pbp2p.Attestation{}}, nil)

	m.proposerClient.EXPECT().ComputeStateRoot(
		gomock.Any(), // context
		gomock.AssignableToTypeOf(&pbp2p.BeaconBlock{}),
	).Return(nil /*response*/, errors.New("something bad happened"))

	validator.ProposeBlock(context.Background(), 55, hex.EncodeToString(validatorKey.PublicKey.Marshal()))
	testutil.AssertLogsContain(t, hook, "something bad happened")
}

func TestProposeBlock_UsesComputedState(t *testing.T) {
	validator, m, finish := setup(t)
	defer finish()

	m.beaconClient.EXPECT().CanonicalHead(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pbp2p.BeaconBlock{}, nil /*err*/)

	m.beaconClient.EXPECT().PendingDeposits(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pb.PendingDepositsResponse{}, nil /*err*/)

	m.beaconClient.EXPECT().Eth1Data(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pb.Eth1DataResponse{}, nil /*err*/)

	m.beaconClient.EXPECT().ForkData(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pbp2p.Fork{
		Epoch:           params.BeaconConfig().GenesisEpoch,
		CurrentVersion:  0,
		PreviousVersion: 0,
	}, nil /*err*/)

	m.proposerClient.EXPECT().PendingAttestations(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pb.PendingAttestationsRequest{}),
	).Return(&pb.PendingAttestationsResponse{PendingAttestations: []*pbp2p.Attestation{}}, nil)

	var broadcastedBlock *pbp2p.BeaconBlock
	m.proposerClient.EXPECT().ProposeBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pbp2p.BeaconBlock{}),
	).Do(func(_ context.Context, blk *pbp2p.BeaconBlock) {
		broadcastedBlock = blk
	}).Return(&pb.ProposeResponse{}, nil /*error*/)

	computedStateRoot := []byte{'T', 'E', 'S', 'T'}
	m.proposerClient.EXPECT().ComputeStateRoot(
		gomock.Any(), // context
		gomock.AssignableToTypeOf(&pbp2p.BeaconBlock{}),
	).Return(
		&pb.StateRootResponse{
			StateRoot: computedStateRoot,
		},
		nil, // err
	)

	validator.ProposeBlock(context.Background(), 55, hex.EncodeToString(validatorKey.PublicKey.Marshal()))

	if !bytes.Equal(broadcastedBlock.StateRootHash32, computedStateRoot) {
		t.Errorf("Unexpected state root hash. want=%#x got=%#x", computedStateRoot, broadcastedBlock.StateRootHash32)
	}
}

func TestProposeBlock_BroadcastsABlock(t *testing.T) {
	validator, m, finish := setup(t)
	defer finish()

	m.beaconClient.EXPECT().CanonicalHead(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pbp2p.BeaconBlock{}, nil /*err*/)

	m.beaconClient.EXPECT().PendingDeposits(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pb.PendingDepositsResponse{}, nil /*err*/)

	m.beaconClient.EXPECT().Eth1Data(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pb.Eth1DataResponse{}, nil /*err*/)

	m.beaconClient.EXPECT().ForkData(
		gomock.Any(), // ctx
		gomock.Eq(&ptypes.Empty{}),
	).Return(&pbp2p.Fork{
		Epoch:           params.BeaconConfig().GenesisEpoch,
		CurrentVersion:  0,
		PreviousVersion: 0,
	}, nil /*err*/)

	m.proposerClient.EXPECT().PendingAttestations(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pb.PendingAttestationsRequest{}),
	).Return(&pb.PendingAttestationsResponse{PendingAttestations: []*pbp2p.Attestation{}}, nil)

	m.proposerClient.EXPECT().ComputeStateRoot(
		gomock.Any(), // context
		gomock.AssignableToTypeOf(&pbp2p.BeaconBlock{}),
	).Return(&pb.StateRootResponse{
		StateRoot: []byte{'F'},
	}, nil /*err*/)

	m.proposerClient.EXPECT().ProposeBlock(
		gomock.Any(), // ctx
		gomock.AssignableToTypeOf(&pbp2p.BeaconBlock{}),
	).Return(&pb.ProposeResponse{}, nil /*error*/)

	validator.ProposeBlock(context.Background(), 55, hex.EncodeToString(validatorKey.PublicKey.Marshal()))
}
