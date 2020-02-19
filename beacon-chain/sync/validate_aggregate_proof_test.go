package sync

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsubpb "github.com/libp2p/go-libp2p-pubsub/pb"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
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
	if err := s.SetSlot(params.BeaconConfig().SlotsPerEpoch); err != nil {
		t.Fatal(err)
	}

	bf := []byte{0xff}
	att := &ethpb.Attestation{Data: &ethpb.AttestationData{
		Target: &ethpb.Checkpoint{Epoch: 0}},
		AggregationBits: bf}

	committee, err := helpers.BeaconCommitteeFromState(s, att.Data.Slot, att.Data.CommitteeIndex)
	if err != nil {
		t.Error(err)
	}
	indices, err := attestationutil.AttestingIndices(att.AggregationBits, committee)
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
	domain := helpers.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainBeaconAttester)
	sig := privKeys[0].Sign(slotRoot[:], domain)

	if err := validateSelection(ctx, beaconState, data, 0, sig.Marshal()); err != nil {
		t.Fatal(err)
	}
}

func TestValidateAggregateAndProof_NoBlock(t *testing.T) {
	db := dbtest.SetupDB(t)
	defer dbtest.TeardownDB(t, db)
	p := p2ptest.NewTestP2P(t)

	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
			Target: &ethpb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
		},
	}

	aggregateAndProof := &ethpb.AggregateAttestationAndProof{
		SelectionProof:  []byte{'A'},
		Aggregate:       att,
		AggregatorIndex: 0,
	}

	r := &Service{
		p2p:                  p,
		db:                   db,
		initialSync:          &mockSync.Sync{IsSyncing: false},
		attPool:              attestations.NewPool(),
		blkRootToPendingAtts: make(map[[32]byte][]*ethpb.AggregateAttestationAndProof),
	}

	buf := new(bytes.Buffer)
	if _, err := p.Encoding().Encode(buf, aggregateAndProof); err != nil {
		t.Fatal(err)
	}

	msg := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(aggregateAndProof)],
			},
		},
	}

	if r.validateAggregateAndProof(context.Background(), "", msg) {
		t.Error("Expected validate to fail")
	}
}

func TestValidateAggregateAndProof_NotWithinSlotRange(t *testing.T) {
	db := dbtest.SetupDB(t)
	defer dbtest.TeardownDB(t, db)
	p := p2ptest.NewTestP2P(t)

	validators := uint64(256)
	beaconState, _ := testutil.DeterministicGenesisState(t, validators)

	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	db.SaveBlock(context.Background(), b)
	root, _ := ssz.HashTreeRoot(b.Block)
	s, _ := beaconstate.InitializeFromProto(&pb.BeaconState{})
	db.SaveState(context.Background(), s, root)

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

	aggregateAndProof := &ethpb.AggregateAttestationAndProof{
		Aggregate: att,
	}

	if err := beaconState.SetGenesisTime(uint64(time.Now().Unix())); err != nil {
		t.Fatal(err)
	}
	r := &Service{
		p2p:         p,
		db:          db,
		initialSync: &mockSync.Sync{IsSyncing: false},
		chain: &mock.ChainService{Genesis: time.Now(),
			State: beaconState},
		attPool: attestations.NewPool(),
	}

	buf := new(bytes.Buffer)
	if _, err := p.Encoding().Encode(buf, aggregateAndProof); err != nil {
		t.Fatal(err)
	}

	msg := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(aggregateAndProof)],
			},
		},
	}

	if r.validateAggregateAndProof(context.Background(), "", msg) {
		t.Error("Expected validate to fail")
	}

	att.Data.Slot = 1<<64 - 1

	buf = new(bytes.Buffer)
	if _, err := p.Encoding().Encode(buf, aggregateAndProof); err != nil {
		t.Fatal(err)
	}

	msg = &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(aggregateAndProof)],
			},
		},
	}
	if r.validateAggregateAndProof(context.Background(), "", msg) {
		t.Error("Expected validate to fail")
	}
}

func TestValidateAggregateAndProof_ExistedInPool(t *testing.T) {
	db := dbtest.SetupDB(t)
	defer dbtest.TeardownDB(t, db)
	p := p2ptest.NewTestP2P(t)

	validators := uint64(256)
	beaconState, _ := testutil.DeterministicGenesisState(t, validators)

	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	db.SaveBlock(context.Background(), b)
	root, _ := ssz.HashTreeRoot(b.Block)

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

	aggregateAndProof := &ethpb.AggregateAttestationAndProof{
		Aggregate: att,
	}

	if err := beaconState.SetGenesisTime(uint64(time.Now().Unix())); err != nil {
		t.Fatal(err)
	}
	r := &Service{
		attPool:     attestations.NewPool(),
		p2p:         p,
		db:          db,
		initialSync: &mockSync.Sync{IsSyncing: false},
		chain: &mock.ChainService{Genesis: time.Now(),
			State: beaconState},
	}

	buf := new(bytes.Buffer)
	if _, err := p.Encoding().Encode(buf, aggregateAndProof); err != nil {
		t.Fatal(err)
	}

	msg := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(aggregateAndProof)],
			},
		},
	}

	if err := r.attPool.SaveBlockAttestation(att); err != nil {
		t.Fatal(err)
	}
	if r.validateAggregateAndProof(context.Background(), "", msg) {
		t.Error("Expected validate to fail")
	}
}

func TestValidateAggregateAndProof_CanValidate(t *testing.T) {
	db := dbtest.SetupDB(t)
	defer dbtest.TeardownDB(t, db)
	p := p2ptest.NewTestP2P(t)

	validators := uint64(256)
	beaconState, privKeys := testutil.DeterministicGenesisState(t, validators)

	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	db.SaveBlock(context.Background(), b)
	root, _ := ssz.HashTreeRoot(b.Block)
	s, _ := beaconstate.InitializeFromProto(&pb.BeaconState{})
	db.SaveState(context.Background(), s, root)

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

	committee, err := helpers.BeaconCommitteeFromState(beaconState, att.Data.Slot, att.Data.CommitteeIndex)
	if err != nil {
		t.Error(err)
	}
	attestingIndices, err := attestationutil.AttestingIndices(att.AggregationBits, committee)
	if err != nil {
		t.Error(err)
	}
	hashTreeRoot, err := ssz.HashTreeRoot(att.Data)
	if err != nil {
		t.Error(err)
	}
	domain := helpers.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainBeaconAttester)
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
	aggregateAndProof := &ethpb.AggregateAttestationAndProof{
		SelectionProof:  sig.Marshal(),
		Aggregate:       att,
		AggregatorIndex: 154,
	}

	if err := beaconState.SetGenesisTime(uint64(time.Now().Unix())); err != nil {
		t.Fatal(err)
	}
	r := &Service{
		p2p:         p,
		db:          db,
		initialSync: &mockSync.Sync{IsSyncing: false},
		chain: &mock.ChainService{Genesis: time.Now(),
			State:            beaconState,
			ValidAttestation: true,
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			}},
		attPool: attestations.NewPool(),
	}

	buf := new(bytes.Buffer)
	if _, err := p.Encoding().Encode(buf, aggregateAndProof); err != nil {
		t.Fatal(err)
	}

	msg := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(aggregateAndProof)],
			},
		},
	}

	if !r.validateAggregateAndProof(context.Background(), "", msg) {
		t.Fatal("Validated status is false")
	}

	if msg.ValidatorData == nil {
		t.Error("Did not set validator data")
	}
}
