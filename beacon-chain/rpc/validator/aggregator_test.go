package validator

import (
	"context"
	"reflect"
	"strings"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	dbutil "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	mockp2p "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	attaggregation "github.com/prysmaticlabs/prysm/shared/aggregation/attestations"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestSubmitAggregateAndProof_Syncing(t *testing.T) {
	db := dbutil.SetupDB(t)
	ctx := context.Background()

	s := &beaconstate.BeaconState{}

	aggregatorServer := &Server{
		HeadFetcher: &mock.ChainService{State: s},
		SyncChecker: &mockSync.Sync{IsSyncing: true},
		BeaconDB:    db,
	}

	req := &ethpb.AggregateSelectionRequest{CommitteeIndex: 1}
	wanted := "Syncing to latest head, not ready to respond"
	if _, err := aggregatorServer.SubmitAggregateSelectionProof(ctx, req); err == nil || !strings.Contains(err.Error(), wanted) {
		t.Error("Did not receive wanted error")
	}
}

func TestSubmitAggregateAndProof_CantFindValidatorIndex(t *testing.T) {
	db := dbutil.SetupDB(t)
	ctx := context.Background()

	s, err := beaconstate.InitializeFromProto(&pbp2p.BeaconState{
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})
	if err != nil {
		t.Fatal(err)
	}

	server := &Server{
		HeadFetcher: &mock.ChainService{State: s},
		SyncChecker: &mockSync.Sync{IsSyncing: false},
		BeaconDB:    db,
	}

	priv := bls.RandKey()
	sig := priv.Sign([]byte{'A'})
	req := &ethpb.AggregateSelectionRequest{CommitteeIndex: 1, SlotSignature: sig.Marshal(), PublicKey: pubKey(3)}
	wanted := "Could not locate validator index in DB"
	if _, err := server.SubmitAggregateSelectionProof(ctx, req); err == nil || !strings.Contains(err.Error(), wanted) {
		t.Errorf("Did not get wanted error")
	}
}

func TestSubmitAggregateAndProof_IsAggregatorAndNoAtts(t *testing.T) {
	db := dbutil.SetupDB(t)
	ctx := context.Background()

	s, err := beaconstate.InitializeFromProto(&pbp2p.BeaconState{
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		Validators: []*ethpb.Validator{
			{PublicKey: pubKey(0)},
			{PublicKey: pubKey(1)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	server := &Server{
		HeadFetcher: &mock.ChainService{State: s},
		SyncChecker: &mockSync.Sync{IsSyncing: false},
		BeaconDB:    db,
		AttPool:     attestations.NewPool(),
	}

	priv := bls.RandKey()
	sig := priv.Sign([]byte{'A'})
	v, err := s.ValidatorAtIndex(1)
	if err != nil {
		t.Fatal(err)
	}
	pubKey := v.PublicKey
	req := &ethpb.AggregateSelectionRequest{CommitteeIndex: 1, SlotSignature: sig.Marshal(), PublicKey: pubKey}

	if _, err := server.SubmitAggregateSelectionProof(ctx, req); err == nil || !strings.Contains(err.Error(), "Could not find attestation for slot and committee in pool") {
		t.Error("Did not get wanted error")
	}
}

func TestSubmitAggregateAndProof_UnaggregateOk(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	c := params.MinimalSpecConfig()
	c.TargetAggregatorsPerCommittee = 16
	params.OverrideBeaconConfig(c)

	db := dbutil.SetupDB(t)
	ctx := context.Background()

	beaconState, privKeys := testutil.DeterministicGenesisState(t, 32)
	att0, err := generateUnaggregatedAtt(beaconState, 0, privKeys)
	if err != nil {
		t.Fatal(err)
	}
	err = beaconState.SetSlot(beaconState.Slot() + params.BeaconConfig().MinAttestationInclusionDelay)
	if err != nil {
		t.Fatal(err)
	}

	aggregatorServer := &Server{
		HeadFetcher: &mock.ChainService{State: beaconState},
		SyncChecker: &mockSync.Sync{IsSyncing: false},
		BeaconDB:    db,
		AttPool:     attestations.NewPool(),
		P2P:         &mockp2p.MockBroadcaster{},
	}

	priv := bls.RandKey()
	sig := priv.Sign([]byte{'B'})
	v, err := beaconState.ValidatorAtIndex(1)
	if err != nil {
		t.Fatal(err)
	}
	pubKey := v.PublicKey
	req := &ethpb.AggregateSelectionRequest{CommitteeIndex: 1, SlotSignature: sig.Marshal(), PublicKey: pubKey}

	if err := aggregatorServer.AttPool.SaveUnaggregatedAttestation(att0); err != nil {
		t.Fatal(err)
	}
	if _, err := aggregatorServer.SubmitAggregateSelectionProof(ctx, req); err != nil {
		t.Fatal(err)
	}
}

func TestSubmitAggregateAndProof_AggregateOk(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	c := params.MinimalSpecConfig()
	c.TargetAggregatorsPerCommittee = 16
	params.OverrideBeaconConfig(c)

	db := dbutil.SetupDB(t)
	ctx := context.Background()

	beaconState, privKeys := testutil.DeterministicGenesisState(t, 32)
	att0, err := generateAtt(beaconState, 0, privKeys)
	if err != nil {
		t.Fatal(err)
	}
	att1, err := generateAtt(beaconState, 2, privKeys)
	if err != nil {
		t.Fatal(err)
	}

	err = beaconState.SetSlot(beaconState.Slot() + params.BeaconConfig().MinAttestationInclusionDelay)
	if err != nil {
		t.Fatal(err)
	}

	aggregatorServer := &Server{
		HeadFetcher: &mock.ChainService{State: beaconState},
		SyncChecker: &mockSync.Sync{IsSyncing: false},
		BeaconDB:    db,
		AttPool:     attestations.NewPool(),
		P2P:         &mockp2p.MockBroadcaster{},
	}

	priv := bls.RandKey()
	sig := priv.Sign([]byte{'B'})
	v, err := beaconState.ValidatorAtIndex(1)
	if err != nil {
		t.Fatal(err)
	}
	pubKey := v.PublicKey
	req := &ethpb.AggregateSelectionRequest{CommitteeIndex: 1, SlotSignature: sig.Marshal(), PublicKey: pubKey}

	if err := aggregatorServer.AttPool.SaveAggregatedAttestation(att0); err != nil {
		t.Fatal(err)
	}
	if err := aggregatorServer.AttPool.SaveAggregatedAttestation(att1); err != nil {
		t.Fatal(err)
	}

	if _, err := aggregatorServer.SubmitAggregateSelectionProof(ctx, req); err != nil {
		t.Fatal(err)
	}

	aggregatedAtts := aggregatorServer.AttPool.AggregatedAttestations()
	wanted, err := attaggregation.AggregatePair(att0, att1)
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(aggregatedAtts, wanted) {
		t.Error("Did not receive wanted attestation")
	}
}

func TestSubmitAggregateAndProof_AggregateNotOk(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	c := params.MinimalSpecConfig()
	c.TargetAggregatorsPerCommittee = 16
	params.OverrideBeaconConfig(c)

	db := dbutil.SetupDB(t)
	ctx := context.Background()

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	if err := beaconState.SetSlot(beaconState.Slot() + params.BeaconConfig().MinAttestationInclusionDelay); err != nil {
		t.Fatal(err)
	}

	aggregatorServer := &Server{
		HeadFetcher: &mock.ChainService{State: beaconState},
		SyncChecker: &mockSync.Sync{IsSyncing: false},
		BeaconDB:    db,
		AttPool:     attestations.NewPool(),
		P2P:         &mockp2p.MockBroadcaster{},
	}

	priv := bls.RandKey()
	sig := priv.Sign([]byte{'B'})
	v, err := beaconState.ValidatorAtIndex(1)
	if err != nil {
		t.Fatal(err)
	}
	pubKey := v.PublicKey
	req := &ethpb.AggregateSelectionRequest{CommitteeIndex: 1, SlotSignature: sig.Marshal(), PublicKey: pubKey}

	if _, err := aggregatorServer.SubmitAggregateSelectionProof(ctx, req); !strings.Contains(err.Error(), "Could not find attestation for slot and committee in pool") {
		t.Error("Did not get wanted error")
	}

	aggregatedAtts := aggregatorServer.AttPool.AggregatedAttestations()
	if len(aggregatedAtts) != 0 {
		t.Errorf("Wanted aggregated attestation 0, got %d", len(aggregatedAtts))
	}
}

func generateAtt(state *beaconstate.BeaconState, index uint64, privKeys []*bls.SecretKey) (*ethpb.Attestation, error) {
	aggBits := bitfield.NewBitlist(4)
	aggBits.SetBitAt(index, true)
	aggBits.SetBitAt(index+1, true)
	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			CommitteeIndex: 1,
			Source:         &ethpb.Checkpoint{Epoch: 0, Root: params.BeaconConfig().ZeroHash[:]},
			Target:         &ethpb.Checkpoint{Epoch: 0},
		},
		AggregationBits: aggBits,
	}
	committee, err := helpers.BeaconCommitteeFromState(state, att.Data.Slot, att.Data.CommitteeIndex)
	if err != nil {
		return nil, err
	}
	attestingIndices := attestationutil.AttestingIndices(att.AggregationBits, committee)
	domain, err := helpers.Domain(state.Fork(), 0, params.BeaconConfig().DomainBeaconAttester, params.BeaconConfig().ZeroHash[:])
	if err != nil {
		return nil, err
	}

	sigs := make([]*bls.Signature, len(attestingIndices))
	zeroSig := [96]byte{}
	att.Signature = zeroSig[:]

	for i, indice := range attestingIndices {
		hashTreeRoot, err := helpers.ComputeSigningRoot(att.Data, domain)
		if err != nil {
			return nil, err
		}
		sig := privKeys[indice].Sign(hashTreeRoot[:])
		sigs[i] = sig
	}

	att.Signature = bls.AggregateSignatures(sigs).Marshal()[:]

	return att, nil
}

func generateUnaggregatedAtt(state *beaconstate.BeaconState, index uint64, privKeys []*bls.SecretKey) (*ethpb.Attestation, error) {
	aggBits := bitfield.NewBitlist(4)
	aggBits.SetBitAt(index, true)
	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			CommitteeIndex: 1,
			Source:         &ethpb.Checkpoint{Epoch: 0, Root: params.BeaconConfig().ZeroHash[:]},
			Target:         &ethpb.Checkpoint{Epoch: 0},
		},
		AggregationBits: aggBits,
	}
	committee, err := helpers.BeaconCommitteeFromState(state, att.Data.Slot, att.Data.CommitteeIndex)
	if err != nil {
		return nil, err
	}
	attestingIndices := attestationutil.AttestingIndices(att.AggregationBits, committee)
	domain, err := helpers.Domain(state.Fork(), 0, params.BeaconConfig().DomainBeaconAttester, params.BeaconConfig().ZeroHash[:])
	if err != nil {
		return nil, err
	}

	sigs := make([]*bls.Signature, len(attestingIndices))
	zeroSig := [96]byte{}
	att.Signature = zeroSig[:]

	for i, indice := range attestingIndices {
		hashTreeRoot, err := helpers.ComputeSigningRoot(att.Data, domain)
		if err != nil {
			return nil, err
		}
		sig := privKeys[indice].Sign(hashTreeRoot[:])
		sigs[i] = sig
	}

	att.Signature = bls.AggregateSignatures(sigs).Marshal()[:]

	return att, nil
}
