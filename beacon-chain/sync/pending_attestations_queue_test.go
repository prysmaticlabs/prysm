package sync

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/libp2p/go-libp2p-core/network"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/peers"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestProcessPendingAtts_NoBlockRequestBlock(t *testing.T) {
	hook := logTest.NewGlobal()
	db, _ := dbtest.SetupDB(t)
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	if len(p1.BHost.Network().Peers()) != 1 {
		t.Error("Expected peers to be connected")
	}
	p1.Peers().Add(new(enr.Record), p2.PeerID(), nil, network.DirOutbound)
	p1.Peers().SetConnectionState(p2.PeerID(), peers.PeerConnected)
	p1.Peers().SetChainState(p2.PeerID(), &pb.Status{})

	r := &Service{
		p2p:                  p1,
		db:                   db,
		chain:                &mock.ChainService{Genesis: roughtime.Now()},
		blkRootToPendingAtts: make(map[[32]byte][]*ethpb.SignedAggregateAttestationAndProof),
		stateSummaryCache:    cache.NewStateSummaryCache(),
	}

	a := &ethpb.AggregateAttestationAndProof{Aggregate: &ethpb.Attestation{Data: &ethpb.AttestationData{Target: &ethpb.Checkpoint{}}}}
	r.blkRootToPendingAtts[[32]byte{'A'}] = []*ethpb.SignedAggregateAttestationAndProof{{Message: a}}
	if err := r.processPendingAtts(context.Background()); err != nil {
		t.Fatal(err)
	}

	testutil.AssertLogsContain(t, hook, "Requesting block for pending attestation")
}

func TestProcessPendingAtts_HasBlockSaveUnAggregatedAtt(t *testing.T) {
	hook := logTest.NewGlobal()
	db, _ := dbtest.SetupDB(t)
	p1 := p2ptest.NewTestP2P(t)
	resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{NewStateMgmt: true})
	defer resetCfg()

	r := &Service{
		p2p:                  p1,
		db:                   db,
		chain:                &mock.ChainService{Genesis: roughtime.Now()},
		blkRootToPendingAtts: make(map[[32]byte][]*ethpb.SignedAggregateAttestationAndProof),
		attPool:              attestations.NewPool(),
		stateSummaryCache:    cache.NewStateSummaryCache(),
	}

	a := &ethpb.AggregateAttestationAndProof{
		Aggregate: &ethpb.Attestation{
			Signature:       bls.RandKey().Sign([]byte("foo")).Marshal(),
			AggregationBits: bitfield.Bitlist{0x02},
			Data: &ethpb.AttestationData{
				Target: &ethpb.Checkpoint{}}}}

	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	r32, err := stateutil.BlockRoot(b.Block)
	if err != nil {
		t.Fatal(err)
	}
	s := testutil.NewBeaconState()
	if err := r.db.SaveBlock(context.Background(), b); err != nil {
		t.Fatal(err)
	}
	if err := r.db.SaveState(context.Background(), s, r32); err != nil {
		t.Fatal(err)
	}

	r.blkRootToPendingAtts[r32] = []*ethpb.SignedAggregateAttestationAndProof{{Message: a}}
	if err := r.processPendingAtts(context.Background()); err != nil {
		t.Fatal(err)
	}

	if len(r.attPool.UnaggregatedAttestations()) != 1 {
		t.Error("Did not save unaggregated att")
	}
	if !reflect.DeepEqual(r.attPool.UnaggregatedAttestations()[0], a.Aggregate) {
		t.Error("Incorrect saved att")
	}
	if len(r.attPool.AggregatedAttestations()) != 0 {
		t.Error("Did save aggregated att")
	}

	testutil.AssertLogsContain(t, hook, "Verified and saved pending attestations to pool")
}

func TestProcessPendingAtts_HasBlockSaveAggregatedAtt(t *testing.T) {
	hook := logTest.NewGlobal()
	db, _ := dbtest.SetupDB(t)
	p1 := p2ptest.NewTestP2P(t)
	validators := uint64(256)
	testutil.ResetCache()
	beaconState, privKeys := testutil.DeterministicGenesisState(t, validators)

	sb := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	if err := db.SaveBlock(context.Background(), sb); err != nil {
		t.Fatal(err)
	}
	root, err := stateutil.BlockRoot(sb.Block)
	if err != nil {
		t.Fatal(err)
	}

	aggBits := bitfield.NewBitlist(3)
	aggBits.SetBitAt(0, true)
	aggBits.SetBitAt(1, true)
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
	sigs := make([]bls.Signature, len(attestingIndices))
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
	// Arbitrary aggregator index for testing purposes.
	aggregatorIndex := committee[0]
	sig := privKeys[aggregatorIndex].Sign(slotRoot[:])
	aggregateAndProof := &ethpb.AggregateAttestationAndProof{
		SelectionProof:  sig.Marshal(),
		Aggregate:       att,
		AggregatorIndex: aggregatorIndex,
	}
	attesterDomain, err = helpers.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainAggregateAndProof, beaconState.GenesisValidatorRoot())
	if err != nil {
		t.Fatal(err)
	}
	signingRoot, err := helpers.ComputeSigningRoot(aggregateAndProof, attesterDomain)
	if err != nil {
		t.Error(err)
	}
	aggreSig := privKeys[aggregatorIndex].Sign(signingRoot[:]).Marshal()

	if err := beaconState.SetGenesisTime(uint64(time.Now().Unix())); err != nil {
		t.Fatal(err)
	}

	r := &Service{
		p2p: p1,
		db:  db,
		chain: &mock.ChainService{Genesis: time.Now(),
			State: beaconState,
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			}},
		blkRootToPendingAtts: make(map[[32]byte][]*ethpb.SignedAggregateAttestationAndProof),
		attPool:              attestations.NewPool(),
		stateSummaryCache:    cache.NewStateSummaryCache(),
	}

	sb = &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	r32, err := stateutil.BlockRoot(sb.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.db.SaveBlock(context.Background(), sb); err != nil {
		t.Fatal(err)
	}
	s := testutil.NewBeaconState()
	if err := r.db.SaveState(context.Background(), s, r32); err != nil {
		t.Fatal(err)
	}

	r.blkRootToPendingAtts[r32] = []*ethpb.SignedAggregateAttestationAndProof{{Message: aggregateAndProof, Signature: aggreSig}}
	if err := r.processPendingAtts(context.Background()); err != nil {
		t.Fatal(err)
	}

	if len(r.attPool.AggregatedAttestations()) != 1 {
		t.Fatal("Did not save aggregated att")
	}
	if !reflect.DeepEqual(r.attPool.AggregatedAttestations()[0], att) {
		t.Error("Incorrect saved att")
	}
	if len(r.attPool.UnaggregatedAttestations()) != 0 {
		t.Error("Did save unaggregated att")
	}

	testutil.AssertLogsContain(t, hook, "Verified and saved pending attestations to pool")
}

func TestValidatePendingAtts_CanPruneOldAtts(t *testing.T) {
	s := &Service{
		blkRootToPendingAtts: make(map[[32]byte][]*ethpb.SignedAggregateAttestationAndProof),
	}

	// 100 Attestations per block root.
	r1 := [32]byte{'A'}
	r2 := [32]byte{'B'}
	r3 := [32]byte{'C'}

	for i := 0; i < 100; i++ {
		s.savePendingAtt(&ethpb.SignedAggregateAttestationAndProof{
			Message: &ethpb.AggregateAttestationAndProof{
				Aggregate: &ethpb.Attestation{
					Data: &ethpb.AttestationData{Slot: uint64(i), BeaconBlockRoot: r1[:]}}}})
		s.savePendingAtt(&ethpb.SignedAggregateAttestationAndProof{
			Message: &ethpb.AggregateAttestationAndProof{
				Aggregate: &ethpb.Attestation{
					Data: &ethpb.AttestationData{Slot: uint64(i), BeaconBlockRoot: r2[:]}}}})
		s.savePendingAtt(&ethpb.SignedAggregateAttestationAndProof{
			Message: &ethpb.AggregateAttestationAndProof{
				Aggregate: &ethpb.Attestation{
					Data: &ethpb.AttestationData{Slot: uint64(i), BeaconBlockRoot: r3[:]}}}})
	}

	if len(s.blkRootToPendingAtts[r1]) != 100 {
		t.Error("Did not save pending atts")
	}
	if len(s.blkRootToPendingAtts[r2]) != 100 {
		t.Error("Did not save pending atts")
	}
	if len(s.blkRootToPendingAtts[r3]) != 100 {
		t.Error("Did not save pending atts")
	}

	// Set current slot to 50, it should prune 19 attestations. (50 - 31)
	s.validatePendingAtts(context.Background(), 50)
	if len(s.blkRootToPendingAtts[r1]) != 81 {
		t.Error("Did not delete pending atts")
	}
	if len(s.blkRootToPendingAtts[r2]) != 81 {
		t.Error("Did not delete pending atts")
	}
	if len(s.blkRootToPendingAtts[r3]) != 81 {
		t.Error("Did not delete pending atts")
	}

	// Set current slot to 100 + slot_duration, it should prune all the attestations.
	s.validatePendingAtts(context.Background(), 100+params.BeaconConfig().SlotsPerEpoch)
	if len(s.blkRootToPendingAtts[r1]) != 0 {
		t.Error("Did not delete pending atts")
	}
	if len(s.blkRootToPendingAtts[r2]) != 0 {
		t.Error("Did not delete pending atts")
	}
	if len(s.blkRootToPendingAtts[r3]) != 0 {
		t.Error("Did not delete pending atts")
	}

	// Verify the keys are deleted.
	if len(s.blkRootToPendingAtts) != 0 {
		t.Error("Did not delete block keys")
	}
}
