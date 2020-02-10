package slasher

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/slashings"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func init() {
	// Use minimal config to reduce test setup time.
	params.OverrideBeaconConfig(params.MinimalSpecConfig())
	params.BeaconConfig().SecondsPerSlot = 1
}

func TestClient_SlashingPoolFeeder_ContextCanceled(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	s := &beaconstate.BeaconState{}

	slasherClient := &Client{
		HeadFetcher: &mock.ChainService{State: s},
	}

	wanted := "Stream context canceled"
	var err error
	go func() {
		err = slasherClient.SlashingPoolFeeder(ctx)
	}()
	cancel()
	time.Sleep(time.Millisecond)
	if !strings.Contains(err.Error(), wanted) {
		t.Error("Did not receive wanted error")
	}
}

func TestClient_SlashingPoolFeeder_NoSlasher(t *testing.T) {
	ctx := context.Background()
	s := &beaconstate.BeaconState{}

	slasherClient := &Client{
		HeadFetcher: &mock.ChainService{State: s},
	}

	wanted := "Slasher server has not been started"
	var err error
	go func() {
		err = slasherClient.SlashingPoolFeeder(ctx)
	}()
	time.Sleep(time.Second * 9)
	if !strings.Contains(err.Error(), wanted) {
		t.Error("Did not receive wanted error")
	}
}

func TestClient_UpdatePool_FullCycle(t *testing.T) {
	ctx := context.Background()
	// Generate validators and state for the 2 attestations.
	validatorCount := 10
	validators := make([]*ethpb.Validator, validatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	base := &pb.BeaconState{
		Validators:  validators,
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}
	s, err := state.InitializeFromProto(base)
	if err != nil {
		t.Fatal(err)
	}
	slasherClient := &fakeSlasher{}
	wantAtt := make([]*ethpb.AttesterSlashing, 1)
	at1 := &ethpb.AttesterSlashing{
		Attestation_1: &ethpb.IndexedAttestation{AttestingIndices: []uint64{1}},
		Attestation_2: &ethpb.IndexedAttestation{AttestingIndices: []uint64{1}}}
	wantAtt[0] = at1
	slasherClient.attesterSlashing = wantAtt

	wantProSlash := make([]*ethpb.ProposerSlashing, 1)
	ps := &ethpb.ProposerSlashing{ProposerIndex: 1}
	wantProSlash[0] = ps
	slasherClient.proposerSlashings = wantProSlash

	client := &Client{
		HeadFetcher:     &mock.ChainService{State: s},
		SlashingPool:    slashings.NewPool(),
		SlasherClient:   slasherClient,
		ShouldBroadcast: false,
	}

	if err := client.updatePool(ctx); err != nil {
		t.Fatal(err)
	}
	if !slasherClient.ProposerSlashingsCalled {
		t.Fatal("Expected ProposerSlashings() to be called")
	}
	if !slasherClient.AttesterSlashingsCalled {
		t.Fatal("Expected AttesterSlashings() to be called")
	}
	pendAttSlash := client.SlashingPool.PendingAttesterSlashings()
	if !reflect.DeepEqual(pendAttSlash, wantAtt) {
		t.Fatalf("expected pool to be filled with: %v got: %v", wantAtt, pendAttSlash)
	}
	pendProSlash := client.SlashingPool.PendingProposerSlashings()
	if !reflect.DeepEqual(pendProSlash, wantProSlash) {
		t.Fatalf("expected pool to be filled with: %v got: %v", wantAtt, pendAttSlash)
	}
}
