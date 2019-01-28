package rpc

import (
	"context"
	"errors"
	"testing"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestCurrentAssignmentsAndGenesisTime(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	genesis := b.NewGenesisBlock([]byte{})
	if err := db.SaveBlock(genesis); err != nil {
		t.Fatalf("Could not save genesis block: %v", err)
	}
	genesisTime := uint64(time.Now().Unix())
	err := db.InitializeState(genesisTime)
	if err != nil {
		t.Fatalf("Can't initialze genesis state: %v", err)
	}
	beaconState, err := db.State()
	if err != nil {
		t.Fatalf("Can't get genesis state: %v", err)
	}

	if err := db.UpdateChainHead(genesis, beaconState); err != nil {
		t.Fatalf("Could not save genesis state: %v", err)
	}

	beaconServer := &BeaconServer{
		beaconDB: db,
		ctx:      context.Background(),
	}

	pubkey := hashutil.Hash([]byte{byte(0)})
	key := &pb.PublicKey{PublicKey: pubkey[:]}
	publicKeys := []*pb.PublicKey{key}
	req := &pb.ValidatorAssignmentRequest{
		PublicKeys: publicKeys,
	}

	res, err := beaconServer.CurrentAssignmentsAndGenesisTime(context.Background(), req)
	if err != nil {
		t.Errorf("Could not call CurrentAssignments correctly: %v", err)
	}

	genesisTimeStamp, err := ptypes.TimestampProto(time.Unix(int64(beaconState.GenesisTime), 0))
	if err != nil {
		t.Errorf("Could not generate genesis timestamp %v", err)
	}

	if res.GenesisTimestamp.String() != genesisTimeStamp.String() {
		t.Errorf(
			"Received different genesis timestamp, wanted: %v, received: %v",
			genesisTimeStamp.String(),
			res.GenesisTimestamp,
		)
	}
}

func TestLatestAttestationContextClosed(t *testing.T) {
	hook := logTest.NewGlobal()
	mockAttestationService := &mockAttestationService{}
	ctx, cancel := context.WithCancel(context.Background())
	beaconServer := &BeaconServer{
		ctx:                ctx,
		attestationService: mockAttestationService,
	}
	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := internal.NewMockBeaconService_LatestAttestationServer(ctrl)
	go func(tt *testing.T) {
		if err := beaconServer.LatestAttestation(&ptypes.Empty{}, mockStream); err != nil {
			tt.Errorf("Could not call RPC method: %v", err)
		}
		<-exitRoutine
	}(t)
	cancel()
	exitRoutine <- true
	testutil.AssertLogsContain(t, hook, "RPC context closed, exiting goroutine")
}

func TestLatestAttestationFaulty(t *testing.T) {
	attestationService := &mockAttestationService{}
	ctx, cancel := context.WithCancel(context.Background())
	beaconServer := &BeaconServer{
		ctx:                 ctx,
		attestationService:  attestationService,
		incomingAttestation: make(chan *pbp2p.Attestation, 0),
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	exitRoutine := make(chan bool)
	attestation := &pbp2p.Attestation{}

	mockStream := internal.NewMockBeaconService_LatestAttestationServer(ctrl)
	mockStream.EXPECT().Send(attestation).Return(errors.New("something wrong"))
	// Tests a faulty stream.
	go func(tt *testing.T) {
		if err := beaconServer.LatestAttestation(&ptypes.Empty{}, mockStream); err.Error() != "something wrong" {
			tt.Errorf("Faulty stream should throw correct error, wanted 'something wrong', got %v", err)
		}
		<-exitRoutine
	}(t)

	beaconServer.incomingAttestation <- attestation
	cancel()
	exitRoutine <- true
}

func TestLatestAttestation(t *testing.T) {
	hook := logTest.NewGlobal()
	attestationService := &mockAttestationService{}
	ctx, cancel := context.WithCancel(context.Background())
	beaconServer := &BeaconServer{
		ctx:                 ctx,
		attestationService:  attestationService,
		incomingAttestation: make(chan *pbp2p.Attestation, 0),
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	exitRoutine := make(chan bool)
	attestation := &pbp2p.Attestation{}
	mockStream := internal.NewMockBeaconService_LatestAttestationServer(ctrl)
	mockStream.EXPECT().Send(attestation).Return(nil)
	// Tests a good stream.
	go func(tt *testing.T) {
		if err := beaconServer.LatestAttestation(&ptypes.Empty{}, mockStream); err != nil {
			tt.Errorf("Could not call RPC method: %v", err)
		}
		<-exitRoutine
	}(t)
	beaconServer.incomingAttestation <- attestation
	cancel()
	exitRoutine <- true

	testutil.AssertLogsContain(t, hook, "Sending attestation to RPC clients")
}

func TestValidatorAssignments(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	mockChain := newMockChainService()

	genesis := b.NewGenesisBlock([]byte{})
	if err := db.SaveBlock(genesis); err != nil {
		t.Fatalf("Could not save genesis block: %v", err)
	}

	genesisTime := uint64(time.Now().Unix())
	err := db.InitializeState(genesisTime)
	if err != nil {
		t.Fatalf("Can't initialze genesis state: %v", err)
	}
	beaconState, err := db.State()
	if err != nil {
		t.Fatalf("Can't get genesis state: %v", err)
	}

	if err := db.UpdateChainHead(genesis, beaconState); err != nil {
		t.Fatalf("Could not save genesis state: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	beaconServer := &BeaconServer{
		ctx:                ctx,
		chainService:       mockChain,
		beaconDB:           db,
		canonicalStateChan: make(chan *pbp2p.BeaconState, 0),
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStream := internal.NewMockBeaconService_ValidatorAssignmentsServer(ctrl)
	mockStream.EXPECT().Send(gomock.Any()).Return(nil)

	pubkey := hashutil.Hash([]byte{byte(0)})
	key := &pb.PublicKey{PublicKey: pubkey[:]}
	publicKeys := []*pb.PublicKey{key}
	req := &pb.ValidatorAssignmentRequest{
		PublicKeys: publicKeys,
	}

	exitRoutine := make(chan bool)

	// Tests a validator assignment stream.
	go func(tt *testing.T) {
		if err := beaconServer.ValidatorAssignments(req, mockStream); err != nil {
			tt.Errorf("Could not stream validators: %v", err)
		}
		<-exitRoutine
	}(t)

	beaconServer.canonicalStateChan <- beaconState
	cancel()
	exitRoutine <- true
	testutil.AssertLogsContain(t, hook, "Sending new cycle assignments to validator clients")
}

func TestAssignmentsForPublicKeys_emptyPubKey(t *testing.T) {
	pks := []*pb.PublicKey{{}}

	a, err := assignmentsForPublicKeys(pks, nil)
	if err != nil {
		t.Error(err)
	}

	if len(a) > 0 {
		t.Errorf("Expected no assignments, but got %v", a)
	}
}
