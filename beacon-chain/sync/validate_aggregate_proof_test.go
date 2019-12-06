package sync

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestVerifyIndexInCommittee_CanVerify(t *testing.T) {
	ctx := context.Background()
	params.UseMinimalConfig()
	defer params.UseMainnetConfig()

	validators := uint64(64)
	s, _ := testutil.DeterministicGenesisState(t, validators)
	s.Slot = params.BeaconConfig().SlotsPerEpoch

	bf := []byte{0xff}
	att := &ethpb.Attestation{Data: &ethpb.AttestationData{
		Target: &ethpb.Checkpoint{Epoch: 0}},
		AggregationBits: bf}

	indices, err := helpers.AttestingIndices(s, att.Data, att.AggregationBits)
	if err != nil {
		t.Fatal(err)
	}

	if err := validateIndexInCommittee(ctx, s, att, indices[0]); err != nil {
		t.Fatal(err)
	}

	wanted := "validator index 1000 is not within the committee"
	if err := validateIndexInCommittee(ctx, s, att, 1000); !strings.Contains(err.Error(), wanted) {
		t.Error("Did not receive wanted error")
	}
}

func TestVerifySelection_NotAnAggregator(t *testing.T) {
	ctx := context.Background()
	params.UseMinimalConfig()
	defer params.UseMainnetConfig()
	validators := uint64(2048)
	beaconState, privKeys := testutil.DeterministicGenesisState(t, validators)

	sig := privKeys[0].Sign([]byte{}, 0)
	data := &ethpb.AttestationData{}

	wanted := "validator is not an aggregator for slot"
	if err := validateSelection(ctx, beaconState, data, 0, sig.Marshal()); !strings.Contains(err.Error(), wanted) {
		t.Error("Did not receive wanted error")
	}
}

func TestVerifySelection_BadSignature(t *testing.T) {
	ctx := context.Background()
	validators := uint64(256)
	beaconState, privKeys := testutil.DeterministicGenesisState(t, validators)

	sig := privKeys[0].Sign([]byte{}, 0)
	data := &ethpb.AttestationData{}

	wanted := "could not validate slot signature"
	if err := validateSelection(ctx, beaconState, data, 0, sig.Marshal()); !strings.Contains(err.Error(), wanted) {
		t.Error("Did not receive wanted error")
	}
}

func TestVerifySelection_CanVerify(t *testing.T) {
	ctx := context.Background()
	validators := uint64(256)
	beaconState, privKeys := testutil.DeterministicGenesisState(t, validators)

	data := &ethpb.AttestationData{}
	slotRoot, err := ssz.HashTreeRoot(data.Slot)
	if err != nil {
		t.Fatal(err)
	}
	domain := helpers.Domain(beaconState.Fork, 0, params.BeaconConfig().DomainBeaconAttester)
	sig := privKeys[0].Sign(slotRoot[:], domain)

	if err := validateSelection(ctx, beaconState, data, 0, sig.Marshal()); err != nil {
		t.Fatal(err)
	}
}

func TestValidateAggregateAndProof_NoBlock(t *testing.T) {
	db := dbtest.SetupDB(t)
	defer dbtest.TeardownDB(t, db)

	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
			Target: &ethpb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
		},
	}

	aggregateAndProof := &pb.AggregateAndProof{
		SelectionProof:  []byte{'A'},
		Aggregate:       att,
		AggregatorIndex: 0,
	}

	r := &RegularSync{
		db:          db,
		initialSync: &mockSync.Sync{IsSyncing: false},
	}

	wanted := "attestation points to a block which is not in the database"
	if _, err := r.validateAggregateAndProof(context.Background(), aggregateAndProof, &p2ptest.MockBroadcaster{}, false); !strings.Contains(err.Error(), wanted) {
		t.Error("Did not receive wanted error")
	}
}

func TestValidateAggregateAndProof_NotWithinSlotRange(t *testing.T) {
	db := dbtest.SetupDB(t)
	defer dbtest.TeardownDB(t, db)

	validators := uint64(256)
	beaconState, _ := testutil.DeterministicGenesisState(t, validators)

	b := &ethpb.BeaconBlock{}
	db.SaveBlock(context.Background(), b)
	root, _ := ssz.SigningRoot(b)

	aggBits := bitfield.NewBitlist(3)
	aggBits.SetBitAt(0, true)
	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Slot:            1,
			BeaconBlockRoot: root[:],
			Source:          &ethpb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
			Target:          &ethpb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
		},
		AggregationBits: aggBits,
	}

	aggregateAndProof := &pb.AggregateAndProof{
		Aggregate: att,
	}

	beaconState.GenesisTime = uint64(time.Now().Unix())
	r := &RegularSync{
		db:          db,
		initialSync: &mockSync.Sync{IsSyncing: false},
		chain: &mock.ChainService{Genesis: time.Now(),
			State: beaconState},
	}

	wanted := fmt.Sprintf("attestation slot out of range %d <= %d <= %d",
		att.Data.Slot, beaconState.Slot, att.Data.Slot+params.BeaconConfig().AttestationPropagationSlotRange)
	if _, err := r.validateAggregateAndProof(context.Background(), aggregateAndProof, &p2ptest.MockBroadcaster{}, false); !strings.Contains(err.Error(), wanted) {
		t.Error("Did not receive wanted error")
	}

	att.Data.Slot = 1<<64 - 1
	wanted = fmt.Sprintf("attestation slot out of range %d <= %d <= %d",
		att.Data.Slot, beaconState.Slot, att.Data.Slot+params.BeaconConfig().AttestationPropagationSlotRange)
	if _, err := r.validateAggregateAndProof(context.Background(), aggregateAndProof, &p2ptest.MockBroadcaster{}, false); !strings.Contains(err.Error(), wanted) {
		t.Error("Did not receive wanted error")
	}
}

func TestValidateAggregateAndProof_CanValidate(t *testing.T) {
	db := dbtest.SetupDB(t)
	defer dbtest.TeardownDB(t, db)

	validators := uint64(256)
	beaconState, privKeys := testutil.DeterministicGenesisState(t, validators)

	b := &ethpb.BeaconBlock{}
	db.SaveBlock(context.Background(), b)
	root, _ := ssz.SigningRoot(b)

	aggBits := bitfield.NewBitlist(3)
	aggBits.SetBitAt(0, true)
	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			BeaconBlockRoot: root[:],
			Source:          &ethpb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
			Target:          &ethpb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
		},
		AggregationBits: aggBits,
	}

	attestingIndices, err := helpers.AttestingIndices(beaconState, att.Data, att.AggregationBits)
	if err != nil {
		t.Error(err)
	}
	hashTreeRoot, err := ssz.HashTreeRoot(att.Data)
	if err != nil {
		t.Error(err)
	}
	domain := helpers.Domain(beaconState.Fork, 0, params.BeaconConfig().DomainBeaconAttester)
	sigs := make([]*bls.Signature, len(attestingIndices))
	for i, indice := range attestingIndices {
		sig := privKeys[indice].Sign(hashTreeRoot[:], domain)
		sigs[i] = sig
	}
	att.Signature = bls.AggregateSignatures(sigs).Marshal()[:]

	slotRoot, err := ssz.HashTreeRoot(att.Data.Slot)
	if err != nil {
		t.Fatal(err)
	}

	sig := privKeys[154].Sign(slotRoot[:], domain)
	aggregateAndProof := &pb.AggregateAndProof{
		SelectionProof:  sig.Marshal(),
		Aggregate:       att,
		AggregatorIndex: 154,
	}

	beaconState.GenesisTime = uint64(time.Now().Unix())
	r := &RegularSync{
		db:          db,
		initialSync: &mockSync.Sync{IsSyncing: false},
		chain: &mock.ChainService{Genesis: time.Now(),
			State: beaconState,
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			}},
	}

	validated, err := r.validateAggregateAndProof(context.Background(), aggregateAndProof, &p2ptest.MockBroadcaster{}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !validated {
		t.Fatal("Validated status is false")
	}
}
