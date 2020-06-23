package sync

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	lru "github.com/hashicorp/golang-lru"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsubpb "github.com/libp2p/go-libp2p-pubsub/pb"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
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
	indices := attestationutil.AttestingIndices(att.AggregationBits, committee)
	if err != nil {
		t.Fatal(err)
	}

	if err := validateIndexInCommittee(ctx, s, att, indices[0]); err != nil {
		t.Fatal(err)
	}

	wanted := "validator index 1000 is not within the committee"
	if err := validateIndexInCommittee(ctx, s, att, 1000); err == nil || !strings.Contains(err.Error(), wanted) {
		t.Error("Did not receive wanted error")
	}
}

func TestVerifyIndexInCommittee_ExistsInBeaconCommittee(t *testing.T) {
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

	if err := validateIndexInCommittee(ctx, s, att, committee[0]); err != nil {
		t.Fatal(err)
	}

	wanted := "validator index 1000 is not within the committee"
	if err := validateIndexInCommittee(ctx, s, att, 1000); err == nil || !strings.Contains(err.Error(), wanted) {
		t.Error("Did not receive wanted error")
	}
}

func TestVerifySelection_NotAnAggregator(t *testing.T) {
	ctx := context.Background()
	params.UseMinimalConfig()
	defer params.UseMainnetConfig()
	validators := uint64(2048)
	beaconState, privKeys := testutil.DeterministicGenesisState(t, validators)

	sig := privKeys[0].Sign([]byte{'A'})
	data := &ethpb.AttestationData{}

	wanted := "validator is not an aggregator for slot"
	if err := validateSelection(ctx, beaconState, data, 0, sig.Marshal()); err == nil || !strings.Contains(err.Error(), wanted) {
		t.Error("Did not receive wanted error")
	}
}

func TestVerifySelection_BadSignature(t *testing.T) {
	ctx := context.Background()
	validators := uint64(256)
	beaconState, privKeys := testutil.DeterministicGenesisState(t, validators)

	sig := privKeys[0].Sign([]byte{'A'})
	data := &ethpb.AttestationData{}

	wanted := "could not validate slot signature"
	if err := validateSelection(ctx, beaconState, data, 0, sig.Marshal()); err == nil || !strings.Contains(err.Error(), wanted) {
		t.Error("Did not receive wanted error")
	}
}

func TestVerifySelection_CanVerify(t *testing.T) {
	ctx := context.Background()
	validators := uint64(256)
	beaconState, privKeys := testutil.DeterministicGenesisState(t, validators)

	data := &ethpb.AttestationData{}
	domain, err := helpers.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainSelectionProof, beaconState.GenesisValidatorRoot())
	if err != nil {
		t.Fatal(err)
	}
	slotRoot, err := helpers.ComputeSigningRoot(data.Slot, domain)
	if err != nil {
		t.Fatal(err)
	}
	sig := privKeys[0].Sign(slotRoot[:])

	if err := validateSelection(ctx, beaconState, data, 0, sig.Marshal()); err != nil {
		t.Fatal(err)
	}
}

func TestValidateAggregateAndProof_NoBlock(t *testing.T) {
	db, _ := dbtest.SetupDB(t)
	p := p2ptest.NewTestP2P(t)

	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0, Root: bytesutil.PadTo([]byte("hello-world"), 32)},
			Target: &ethpb.Checkpoint{Epoch: 0, Root: bytesutil.PadTo([]byte("hello-world"), 32)},
		},
	}

	aggregateAndProof := &ethpb.AggregateAttestationAndProof{
		SelectionProof:  bytesutil.PadTo([]byte{'A'}, 96),
		Aggregate:       att,
		AggregatorIndex: 0,
	}
	signedAggregateAndProof := &ethpb.SignedAggregateAttestationAndProof{Message: aggregateAndProof}

	c, err := lru.New(10)
	if err != nil {
		t.Fatal(err)
	}
	r := &Service{
		p2p:                  p,
		db:                   db,
		initialSync:          &mockSync.Sync{IsSyncing: false},
		attPool:              attestations.NewPool(),
		blkRootToPendingAtts: make(map[[32]byte][]*ethpb.SignedAggregateAttestationAndProof),
		seenAttestationCache: c,
		stateSummaryCache:    cache.NewStateSummaryCache(),
	}

	buf := new(bytes.Buffer)
	if _, err := p.Encoding().Encode(buf, signedAggregateAndProof); err != nil {
		t.Fatal(err)
	}

	msg := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(signedAggregateAndProof)],
			},
		},
	}

	if r.validateAggregateAndProof(context.Background(), "", msg) == pubsub.ValidationAccept {
		t.Error("Expected validate to fail")
	}
}

func TestValidateAggregateAndProof_NotWithinSlotRange(t *testing.T) {
	db, _ := dbtest.SetupDB(t)
	p := p2ptest.NewTestP2P(t)

	validators := uint64(256)
	beaconState, _ := testutil.DeterministicGenesisState(t, validators)

	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	if err := db.SaveBlock(context.Background(), b); err != nil {
		t.Fatal(err)
	}
	root, err := stateutil.BlockRoot(b.Block)
	if err != nil {
		t.Fatal(err)
	}
	s := testutil.NewBeaconState()
	if err := db.SaveState(context.Background(), s, root); err != nil {
		t.Fatal(err)
	}

	aggBits := bitfield.NewBitlist(3)
	aggBits.SetBitAt(0, true)
	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Slot:            1,
			BeaconBlockRoot: root[:],
			Source:          &ethpb.Checkpoint{Epoch: 0, Root: bytesutil.PadTo([]byte("hello-world"), 32)},
			Target:          &ethpb.Checkpoint{Epoch: 0, Root: bytesutil.PadTo([]byte("hello-world"), 32)},
		},
		AggregationBits: aggBits,
	}

	aggregateAndProof := &ethpb.AggregateAttestationAndProof{
		Aggregate: att,
	}
	signedAggregateAndProof := &ethpb.SignedAggregateAttestationAndProof{Message: aggregateAndProof}

	if err := beaconState.SetGenesisTime(uint64(time.Now().Unix())); err != nil {
		t.Fatal(err)
	}

	c, err := lru.New(10)
	if err != nil {
		t.Fatal(err)
	}
	r := &Service{
		p2p:         p,
		db:          db,
		initialSync: &mockSync.Sync{IsSyncing: false},
		chain: &mock.ChainService{Genesis: time.Now(),
			State: beaconState},
		attPool:              attestations.NewPool(),
		seenAttestationCache: c,
		stateSummaryCache:    cache.NewStateSummaryCache(),
	}

	buf := new(bytes.Buffer)
	if _, err := p.Encoding().Encode(buf, signedAggregateAndProof); err != nil {
		t.Fatal(err)
	}

	msg := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(signedAggregateAndProof)],
			},
		},
	}

	if r.validateAggregateAndProof(context.Background(), "", msg) == pubsub.ValidationAccept {
		t.Error("Expected validate to fail")
	}

	att.Data.Slot = 1<<32 - 1

	buf = new(bytes.Buffer)
	if _, err := p.Encoding().Encode(buf, signedAggregateAndProof); err != nil {
		t.Fatal(err)
	}

	msg = &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(signedAggregateAndProof)],
			},
		},
	}
	if r.validateAggregateAndProof(context.Background(), "", msg) == pubsub.ValidationAccept {
		t.Error("Expected validate to fail")
	}
}

func TestValidateAggregateAndProof_ExistedInPool(t *testing.T) {
	db, _ := dbtest.SetupDB(t)
	p := p2ptest.NewTestP2P(t)

	validators := uint64(256)
	beaconState, _ := testutil.DeterministicGenesisState(t, validators)

	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	if err := db.SaveBlock(context.Background(), b); err != nil {
		t.Fatal(err)
	}
	root, err := stateutil.BlockRoot(b.Block)
	if err != nil {
		t.Fatal(err)
	}

	aggBits := bitfield.NewBitlist(3)
	aggBits.SetBitAt(0, true)
	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Slot:            1,
			BeaconBlockRoot: root[:],
			Source:          &ethpb.Checkpoint{Epoch: 0, Root: bytesutil.PadTo([]byte("hello-world"), 32)},
			Target:          &ethpb.Checkpoint{Epoch: 0, Root: bytesutil.PadTo([]byte("hello-world"), 32)},
		},
		AggregationBits: aggBits,
	}

	aggregateAndProof := &ethpb.AggregateAttestationAndProof{
		Aggregate: att,
	}
	signedAggregateAndProof := &ethpb.SignedAggregateAttestationAndProof{Message: aggregateAndProof}

	if err := beaconState.SetGenesisTime(uint64(time.Now().Unix())); err != nil {
		t.Fatal(err)
	}
	c, err := lru.New(10)
	if err != nil {
		t.Fatal(err)
	}
	r := &Service{
		attPool:     attestations.NewPool(),
		p2p:         p,
		db:          db,
		initialSync: &mockSync.Sync{IsSyncing: false},
		chain: &mock.ChainService{Genesis: time.Now(),
			State: beaconState},
		seenAttestationCache: c,
		blkRootToPendingAtts: make(map[[32]byte][]*ethpb.SignedAggregateAttestationAndProof),
	}

	buf := new(bytes.Buffer)
	if _, err := p.Encoding().Encode(buf, signedAggregateAndProof); err != nil {
		t.Fatal(err)
	}

	msg := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(signedAggregateAndProof)],
			},
		},
	}

	if err := r.attPool.SaveBlockAttestation(att); err != nil {
		t.Fatal(err)
	}
	if r.validateAggregateAndProof(context.Background(), "", msg) == pubsub.ValidationAccept {
		t.Error("Expected validate to fail")
	}
}

func TestValidateAggregateAndProofWithNewStateMgmt_CanValidate(t *testing.T) {
	resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{NewStateMgmt: true})
	defer resetCfg()

	db, _ := dbtest.SetupDB(t)
	p := p2ptest.NewTestP2P(t)

	validators := uint64(256)
	beaconState, privKeys := testutil.DeterministicGenesisState(t, validators)

	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	if err := db.SaveBlock(context.Background(), b); err != nil {
		t.Fatal(err)
	}
	root, err := stateutil.BlockRoot(b.Block)
	if err != nil {
		t.Fatal(err)
	}
	s := testutil.NewBeaconState()
	if err := db.SaveState(context.Background(), s, root); err != nil {
		t.Fatal(err)
	}

	aggBits := bitfield.NewBitlist(3)
	aggBits.SetBitAt(0, true)
	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			BeaconBlockRoot: root[:],
			Source:          &ethpb.Checkpoint{Epoch: 0, Root: bytesutil.PadTo([]byte("hello-world"), 32)},
			Target:          &ethpb.Checkpoint{Epoch: 0, Root: bytesutil.PadTo([]byte("hello-world"), 32)},
		},
		AggregationBits: aggBits,
	}

	committee, err := helpers.BeaconCommitteeFromState(beaconState, att.Data.Slot, att.Data.CommitteeIndex)
	if err != nil {
		t.Error(err)
	}
	attestingIndices := attestationutil.AttestingIndices(att.AggregationBits, committee)
	if err != nil {
		t.Error(err)
	}
	attesterDomain, err := helpers.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainBeaconAttester, beaconState.GenesisValidatorRoot())
	if err != nil {
		t.Fatal(err)
	}
	hashTreeRoot, err := helpers.ComputeSigningRoot(att.Data, attesterDomain)
	if err != nil {
		t.Error(err)
	}
	sigs := make([]*bls.Signature, len(attestingIndices))
	for i, indice := range attestingIndices {
		sig := privKeys[indice].Sign(hashTreeRoot[:])
		sigs[i] = sig
	}
	att.Signature = bls.AggregateSignatures(sigs).Marshal()[:]

	selectionDomain, err := helpers.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainSelectionProof, beaconState.GenesisValidatorRoot())
	if err != nil {
		t.Fatal(err)
	}
	slotRoot, err := helpers.ComputeSigningRoot(att.Data.Slot, selectionDomain)
	if err != nil {
		t.Fatal(err)
	}

	sig := privKeys[22].Sign(slotRoot[:])
	aggregateAndProof := &ethpb.AggregateAttestationAndProof{
		SelectionProof:  sig.Marshal(),
		Aggregate:       att,
		AggregatorIndex: 22,
	}
	signedAggregateAndProof := &ethpb.SignedAggregateAttestationAndProof{Message: aggregateAndProof}

	attesterDomain, err = helpers.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainAggregateAndProof, beaconState.GenesisValidatorRoot())
	if err != nil {
		t.Fatal(err)
	}
	signingRoot, err := helpers.ComputeSigningRoot(signedAggregateAndProof.Message, attesterDomain)
	if err != nil {
		t.Error(err)
	}
	aggreSig := privKeys[22].Sign(signingRoot[:]).Marshal()
	signedAggregateAndProof.Signature = aggreSig[:]

	if err := beaconState.SetGenesisTime(uint64(time.Now().Unix())); err != nil {
		t.Fatal(err)
	}
	c, err := lru.New(10)
	if err != nil {
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
		attPool:              attestations.NewPool(),
		seenAttestationCache: c,
		stateSummaryCache:    cache.NewStateSummaryCache(),
	}

	buf := new(bytes.Buffer)
	if _, err := p.Encoding().Encode(buf, signedAggregateAndProof); err != nil {
		t.Fatal(err)
	}

	msg := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(signedAggregateAndProof)],
			},
		},
	}

	if r.validateAggregateAndProof(context.Background(), "", msg) != pubsub.ValidationAccept {
		t.Fatal("Validated status is false")
	}

	if msg.ValidatorData == nil {
		t.Error("Did not set validator data")
	}
}

func TestVerifyIndexInCommittee_SeenAggregatorEpoch(t *testing.T) {
	db, _ := dbtest.SetupDB(t)
	p := p2ptest.NewTestP2P(t)

	validators := uint64(256)
	beaconState, privKeys := testutil.DeterministicGenesisState(t, validators)

	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	if err := db.SaveBlock(context.Background(), b); err != nil {
		t.Fatal(err)
	}
	root, err := stateutil.BlockRoot(b.Block)
	if err != nil {
		t.Fatal(err)
	}
	s := testutil.NewBeaconState()
	if err := db.SaveState(context.Background(), s, root); err != nil {
		t.Fatal(err)
	}

	aggBits := bitfield.NewBitlist(3)
	aggBits.SetBitAt(0, true)
	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			BeaconBlockRoot: root[:],
			Source:          &ethpb.Checkpoint{Epoch: 0, Root: bytesutil.PadTo([]byte("hello-world"), 32)},
			Target:          &ethpb.Checkpoint{Epoch: 0, Root: bytesutil.PadTo([]byte("hello-world"), 32)},
		},
		AggregationBits: aggBits,
	}

	committee, err := helpers.BeaconCommitteeFromState(beaconState, att.Data.Slot, att.Data.CommitteeIndex)
	if err != nil {
		t.Error(err)
	}
	attestingIndices := attestationutil.AttestingIndices(att.AggregationBits, committee)
	attesterDomain, err := helpers.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainBeaconAttester, beaconState.GenesisValidatorRoot())
	if err != nil {
		t.Fatal(err)
	}
	hashTreeRoot, err := helpers.ComputeSigningRoot(att.Data, attesterDomain)
	if err != nil {
		t.Error(err)
	}
	sigs := make([]*bls.Signature, len(attestingIndices))
	for i, indice := range attestingIndices {
		sig := privKeys[indice].Sign(hashTreeRoot[:])
		sigs[i] = sig
	}
	att.Signature = bls.AggregateSignatures(sigs).Marshal()[:]

	selectionDomain, err := helpers.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainSelectionProof, beaconState.GenesisValidatorRoot())
	if err != nil {
		t.Fatal(err)
	}
	slotRoot, err := helpers.ComputeSigningRoot(att.Data.Slot, selectionDomain)
	if err != nil {
		t.Fatal(err)
	}

	sig := privKeys[22].Sign(slotRoot[:])
	aggregateAndProof := &ethpb.AggregateAttestationAndProof{
		SelectionProof:  sig.Marshal(),
		Aggregate:       att,
		AggregatorIndex: 22,
	}
	signedAggregateAndProof := &ethpb.SignedAggregateAttestationAndProof{Message: aggregateAndProof}

	attesterDomain, err = helpers.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainAggregateAndProof, beaconState.GenesisValidatorRoot())
	if err != nil {
		t.Fatal(err)
	}
	signingRoot, err := helpers.ComputeSigningRoot(signedAggregateAndProof.Message, attesterDomain)
	if err != nil {
		t.Error(err)
	}
	aggreSig := privKeys[22].Sign(signingRoot[:]).Marshal()
	signedAggregateAndProof.Signature = aggreSig[:]

	if err := beaconState.SetGenesisTime(uint64(time.Now().Unix())); err != nil {
		t.Fatal(err)
	}

	c, err := lru.New(10)
	if err != nil {
		t.Fatal(err)
	}
	r := &Service{
		p2p:         p,
		db:          db,
		initialSync: &mockSync.Sync{IsSyncing: false},
		chain: &mock.ChainService{Genesis: time.Now(),
			ValidatorsRoot:   [32]byte{'A'},
			State:            beaconState,
			ValidAttestation: true,
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			}},

		attPool:              attestations.NewPool(),
		seenAttestationCache: c,
		stateSummaryCache:    cache.NewStateSummaryCache(),
	}

	buf := new(bytes.Buffer)
	if _, err := p.Encoding().Encode(buf, signedAggregateAndProof); err != nil {
		t.Fatal(err)
	}

	msg := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(signedAggregateAndProof)],
			},
		},
	}

	if r.validateAggregateAndProof(context.Background(), "", msg) != pubsub.ValidationAccept {
		t.Fatal("Validated status is false")
	}

	// Should fail with another attestation in the same epoch.
	signedAggregateAndProof.Message.Aggregate.Data.Slot++
	buf = new(bytes.Buffer)
	if _, err := p.Encoding().Encode(buf, signedAggregateAndProof); err != nil {
		t.Fatal(err)
	}
	msg = &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(signedAggregateAndProof)],
			},
		},
	}

	time.Sleep(10 * time.Millisecond) // Wait for cached value to pass through buffers.
	if r.validateAggregateAndProof(context.Background(), "", msg) == pubsub.ValidationAccept {
		t.Fatal("Validated status is true")
	}
}
