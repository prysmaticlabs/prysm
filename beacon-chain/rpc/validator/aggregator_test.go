package validator

import (
	"context"
	"reflect"
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
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestSubmitAggregateAndProof_Syncing(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()

	s := &beaconstate.BeaconState{}

	aggregatorServer := &Server{
		HeadFetcher: &mock.ChainService{State: s},
		SyncChecker: &mockSync.Sync{IsSyncing: true},
		BeaconDB:    db,
	}

	req := &ethpb.AggregateSelectionRequest{CommitteeIndex: 1}
	wanted := "Syncing to latest head, not ready to respond"
	_, err := aggregatorServer.SubmitAggregateSelectionProof(ctx, req)
	assert.ErrorContains(t, wanted, err)
}

func TestSubmitAggregateAndProof_CantFindValidatorIndex(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()

	s, err := beaconstate.InitializeFromProto(&pbp2p.BeaconState{
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})
	require.NoError(t, err)

	server := &Server{
		HeadFetcher: &mock.ChainService{State: s},
		SyncChecker: &mockSync.Sync{IsSyncing: false},
		BeaconDB:    db,
	}

	priv := bls.RandKey()
	sig := priv.Sign([]byte{'A'})
	req := &ethpb.AggregateSelectionRequest{CommitteeIndex: 1, SlotSignature: sig.Marshal(), PublicKey: pubKey(3)}
	wanted := "Could not locate validator index in DB"
	_, err = server.SubmitAggregateSelectionProof(ctx, req)
	assert.ErrorContains(t, wanted, err)
}

func TestSubmitAggregateAndProof_IsAggregatorAndNoAtts(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()

	s, err := beaconstate.InitializeFromProto(&pbp2p.BeaconState{
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		Validators: []*ethpb.Validator{
			{PublicKey: pubKey(0)},
			{PublicKey: pubKey(1)},
		},
	})
	require.NoError(t, err)

	server := &Server{
		HeadFetcher: &mock.ChainService{State: s},
		SyncChecker: &mockSync.Sync{IsSyncing: false},
		BeaconDB:    db,
		AttPool:     attestations.NewPool(),
	}

	priv := bls.RandKey()
	sig := priv.Sign([]byte{'A'})
	v, err := s.ValidatorAtIndex(1)
	require.NoError(t, err)
	pubKey := v.PublicKey
	req := &ethpb.AggregateSelectionRequest{CommitteeIndex: 1, SlotSignature: sig.Marshal(), PublicKey: pubKey}

	_, err = server.SubmitAggregateSelectionProof(ctx, req)
	assert.ErrorContains(t, "Could not find attestation for slot and committee in pool", err)
}

func TestSubmitAggregateAndProof_UnaggregateOk(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	c := params.MinimalSpecConfig()
	c.TargetAggregatorsPerCommittee = 16
	params.OverrideBeaconConfig(c)

	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()

	beaconState, privKeys := testutil.DeterministicGenesisState(t, 32)
	att0, err := generateUnaggregatedAtt(beaconState, 0, privKeys)
	require.NoError(t, err)
	err = beaconState.SetSlot(beaconState.Slot() + params.BeaconConfig().MinAttestationInclusionDelay)
	require.NoError(t, err)

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
	require.NoError(t, err)
	pubKey := v.PublicKey
	req := &ethpb.AggregateSelectionRequest{CommitteeIndex: 1, SlotSignature: sig.Marshal(), PublicKey: pubKey}

	require.NoError(t, aggregatorServer.AttPool.SaveUnaggregatedAttestation(att0))
	_, err = aggregatorServer.SubmitAggregateSelectionProof(ctx, req)
	require.NoError(t, err)
}

func TestSubmitAggregateAndProof_AggregateOk(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	c := params.MinimalSpecConfig()
	c.TargetAggregatorsPerCommittee = 16
	params.OverrideBeaconConfig(c)

	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()

	beaconState, privKeys := testutil.DeterministicGenesisState(t, 32)
	att0, err := generateAtt(beaconState, 0, privKeys)
	require.NoError(t, err)
	att1, err := generateAtt(beaconState, 2, privKeys)
	require.NoError(t, err)

	err = beaconState.SetSlot(beaconState.Slot() + params.BeaconConfig().MinAttestationInclusionDelay)
	require.NoError(t, err)

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
	require.NoError(t, err)
	pubKey := v.PublicKey
	req := &ethpb.AggregateSelectionRequest{CommitteeIndex: 1, SlotSignature: sig.Marshal(), PublicKey: pubKey}

	require.NoError(t, aggregatorServer.AttPool.SaveAggregatedAttestation(att0))
	require.NoError(t, aggregatorServer.AttPool.SaveAggregatedAttestation(att1))
	_, err = aggregatorServer.SubmitAggregateSelectionProof(ctx, req)
	require.NoError(t, err)

	aggregatedAtts := aggregatorServer.AttPool.AggregatedAttestations()
	wanted, err := attaggregation.AggregatePair(att0, att1)
	require.NoError(t, err)
	if reflect.DeepEqual(aggregatedAtts, wanted) {
		t.Error("Did not receive wanted attestation")
	}
}

func TestSubmitAggregateAndProof_AggregateNotOk(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	c := params.MinimalSpecConfig()
	c.TargetAggregatorsPerCommittee = 16
	params.OverrideBeaconConfig(c)

	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()

	beaconState, _ := testutil.DeterministicGenesisState(t, 32)
	require.NoError(t, beaconState.SetSlot(beaconState.Slot()+params.BeaconConfig().MinAttestationInclusionDelay))

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
	require.NoError(t, err)
	pubKey := v.PublicKey
	req := &ethpb.AggregateSelectionRequest{CommitteeIndex: 1, SlotSignature: sig.Marshal(), PublicKey: pubKey}

	_, err = aggregatorServer.SubmitAggregateSelectionProof(ctx, req)
	assert.ErrorContains(t, "Could not find attestation for slot and committee in pool", err)

	aggregatedAtts := aggregatorServer.AttPool.AggregatedAttestations()
	assert.Equal(t, 0, len(aggregatedAtts), "Wanted aggregated attestation")
}

func generateAtt(state *beaconstate.BeaconState, index uint64, privKeys []bls.SecretKey) (*ethpb.Attestation, error) {
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

	sigs := make([]bls.Signature, len(attestingIndices))
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

func generateUnaggregatedAtt(state *beaconstate.BeaconState, index uint64, privKeys []bls.SecretKey) (*ethpb.Attestation, error) {
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

	sigs := make([]bls.Signature, len(attestingIndices))
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

func TestSubmitAggregateAndProof_PreferOwnAttestation(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	c := params.MinimalSpecConfig()
	c.TargetAggregatorsPerCommittee = 16
	params.OverrideBeaconConfig(c)

	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()

	// This test creates 3 attestations. 0 and 2 have the same attestation data and can be
	// aggregated. 1 has the validator's signature making this request and that is the expected
	// attestation to sign, even though the aggregated 0&2 would have more aggregated bits.
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 32)
	att0, err := generateAtt(beaconState, 0, privKeys)
	require.NoError(t, err)
	att0.Data.BeaconBlockRoot = bytesutil.PadTo([]byte("foo"), 32)
	att0.AggregationBits = bitfield.Bitlist{0b11100}
	att1, err := generateAtt(beaconState, 0, privKeys)
	require.NoError(t, err)
	att1.Data.BeaconBlockRoot = bytesutil.PadTo([]byte("bar"), 32)
	att1.AggregationBits = bitfield.Bitlist{0b11001}
	att2, err := generateAtt(beaconState, 2, privKeys)
	require.NoError(t, err)
	att2.Data.BeaconBlockRoot = bytesutil.PadTo([]byte("foo"), 32)
	att2.AggregationBits = bitfield.Bitlist{0b11110}

	err = beaconState.SetSlot(beaconState.Slot() + params.BeaconConfig().MinAttestationInclusionDelay)
	require.NoError(t, err)

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
	require.NoError(t, err)
	pubKey := v.PublicKey
	req := &ethpb.AggregateSelectionRequest{CommitteeIndex: 1, SlotSignature: sig.Marshal(), PublicKey: pubKey}

	err = aggregatorServer.AttPool.SaveAggregatedAttestations([]*ethpb.Attestation{
		att0,
		att1,
		att2,
	})
	require.NoError(t, err)

	res, err := aggregatorServer.SubmitAggregateSelectionProof(ctx, req)
	require.NoError(t, err)
	assert.DeepEqual(t, att1, res.AggregateAndProof.Aggregate, "Did not receive wanted attestation")
}

func TestSubmitAggregateAndProof_SelectsMostBitsWhenOwnAttestationNotPresent(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	c := params.MinimalSpecConfig()
	c.TargetAggregatorsPerCommittee = 16
	params.OverrideBeaconConfig(c)

	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()

	// This test creates two distinct attestations, neither of which contain the validator's index,
	// index 0. This test should choose the most bits attestation, att1.
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 32)
	att0, err := generateAtt(beaconState, 0, privKeys)
	require.NoError(t, err)
	att0.Data.BeaconBlockRoot = bytesutil.PadTo([]byte("foo"), 32)
	att0.AggregationBits = bitfield.Bitlist{0b11100}
	att1, err := generateAtt(beaconState, 2, privKeys)
	require.NoError(t, err)
	att1.Data.BeaconBlockRoot = bytesutil.PadTo([]byte("bar"), 32)
	att1.AggregationBits = bitfield.Bitlist{0b11110}

	err = beaconState.SetSlot(beaconState.Slot() + params.BeaconConfig().MinAttestationInclusionDelay)
	require.NoError(t, err)

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
	require.NoError(t, err)
	pubKey := v.PublicKey
	req := &ethpb.AggregateSelectionRequest{CommitteeIndex: 1, SlotSignature: sig.Marshal(), PublicKey: pubKey}

	err = aggregatorServer.AttPool.SaveAggregatedAttestations([]*ethpb.Attestation{
		att0,
		att1,
	})
	require.NoError(t, err)

	res, err := aggregatorServer.SubmitAggregateSelectionProof(ctx, req)
	require.NoError(t, err)
	assert.DeepEqual(t, att1, res.AggregateAndProof.Aggregate, "Did not receive wanted attestation")
}
