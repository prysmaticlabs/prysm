package sync

import (
	"context"
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
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestProcessPendingAtts_NoBlockRequestBlock(t *testing.T) {
	hook := logTest.NewGlobal()
	db, _ := dbtest.SetupDB(t)
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	assert.Equal(t, 1, len(p1.BHost.Network().Peers()), "Expected peers to be connected")
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
	require.NoError(t, r.processPendingAtts(context.Background()))
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

	b := testutil.NewBeaconBlock()
	r32, err := stateutil.BlockRoot(b.Block)
	require.NoError(t, err)
	s := testutil.NewBeaconState()
	require.NoError(t, r.db.SaveBlock(context.Background(), b))
	require.NoError(t, r.db.SaveState(context.Background(), s, r32))

	r.blkRootToPendingAtts[r32] = []*ethpb.SignedAggregateAttestationAndProof{{Message: a}}
	require.NoError(t, r.processPendingAtts(context.Background()))

	assert.Equal(t, 1, len(r.attPool.UnaggregatedAttestations()), "Did not save unaggregated att")
	assert.DeepEqual(t, a.Aggregate, r.attPool.UnaggregatedAttestations()[0], "Incorrect saved att")
	assert.Equal(t, 0, len(r.attPool.AggregatedAttestations()), "Did save aggregated att")
	testutil.AssertLogsContain(t, hook, "Verified and saved pending attestations to pool")
}

func TestProcessPendingAtts_HasBlockSaveAggregatedAtt(t *testing.T) {
	hook := logTest.NewGlobal()
	db, _ := dbtest.SetupDB(t)
	p1 := p2ptest.NewTestP2P(t)
	validators := uint64(256)
	testutil.ResetCache()
	beaconState, privKeys := testutil.DeterministicGenesisState(t, validators)

	sb := testutil.NewBeaconBlock()
	require.NoError(t, db.SaveBlock(context.Background(), sb))
	root, err := stateutil.BlockRoot(sb.Block)
	require.NoError(t, err)

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
	assert.NoError(t, err)
	attestingIndices := attestationutil.AttestingIndices(att.AggregationBits, committee)
	assert.NoError(t, err)
	attesterDomain, err := helpers.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainBeaconAttester, beaconState.GenesisValidatorRoot())
	require.NoError(t, err)
	hashTreeRoot, err := helpers.ComputeSigningRoot(att.Data, attesterDomain)
	assert.NoError(t, err)
	sigs := make([]bls.Signature, len(attestingIndices))
	for i, indice := range attestingIndices {
		sig := privKeys[indice].Sign(hashTreeRoot[:])
		sigs[i] = sig
	}
	att.Signature = bls.AggregateSignatures(sigs).Marshal()[:]

	selectionDomain, err := helpers.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainSelectionProof, beaconState.GenesisValidatorRoot())
	require.NoError(t, err)
	slotRoot, err := helpers.ComputeSigningRoot(att.Data.Slot, selectionDomain)
	require.NoError(t, err)
	// Arbitrary aggregator index for testing purposes.
	aggregatorIndex := committee[0]
	sig := privKeys[aggregatorIndex].Sign(slotRoot[:])
	aggregateAndProof := &ethpb.AggregateAttestationAndProof{
		SelectionProof:  sig.Marshal(),
		Aggregate:       att,
		AggregatorIndex: aggregatorIndex,
	}
	attesterDomain, err = helpers.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainAggregateAndProof, beaconState.GenesisValidatorRoot())
	require.NoError(t, err)
	signingRoot, err := helpers.ComputeSigningRoot(aggregateAndProof, attesterDomain)
	assert.NoError(t, err)
	aggreSig := privKeys[aggregatorIndex].Sign(signingRoot[:]).Marshal()

	require.NoError(t, beaconState.SetGenesisTime(uint64(time.Now().Unix())))

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

	sb = testutil.NewBeaconBlock()
	r32, err := stateutil.BlockRoot(sb.Block)
	require.NoError(t, err)
	require.NoError(t, r.db.SaveBlock(context.Background(), sb))
	s := testutil.NewBeaconState()
	require.NoError(t, r.db.SaveState(context.Background(), s, r32))

	r.blkRootToPendingAtts[r32] = []*ethpb.SignedAggregateAttestationAndProof{{Message: aggregateAndProof, Signature: aggreSig}}
	require.NoError(t, r.processPendingAtts(context.Background()))

	assert.Equal(t, 1, len(r.attPool.AggregatedAttestations()), "Did not save aggregated att")
	assert.DeepEqual(t, att, r.attPool.AggregatedAttestations()[0], "Incorrect saved att")
	assert.Equal(t, 0, len(r.attPool.UnaggregatedAttestations()), "Did save unaggregated att")
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

	assert.Equal(t, 100, len(s.blkRootToPendingAtts[r1]), "Did not save pending atts")
	assert.Equal(t, 100, len(s.blkRootToPendingAtts[r2]), "Did not save pending atts")
	assert.Equal(t, 100, len(s.blkRootToPendingAtts[r3]), "Did not save pending atts")

	// Set current slot to 50, it should prune 19 attestations. (50 - 31)
	s.validatePendingAtts(context.Background(), 50)
	assert.Equal(t, 81, len(s.blkRootToPendingAtts[r1]), "Did not delete pending atts")
	assert.Equal(t, 81, len(s.blkRootToPendingAtts[r2]), "Did not delete pending atts")
	assert.Equal(t, 81, len(s.blkRootToPendingAtts[r3]), "Did not delete pending atts")

	// Set current slot to 100 + slot_duration, it should prune all the attestations.
	s.validatePendingAtts(context.Background(), 100+params.BeaconConfig().SlotsPerEpoch)
	assert.Equal(t, 0, len(s.blkRootToPendingAtts[r1]), "Did not delete pending atts")
	assert.Equal(t, 0, len(s.blkRootToPendingAtts[r2]), "Did not delete pending atts")
	assert.Equal(t, 0, len(s.blkRootToPendingAtts[r3]), "Did not delete pending atts")

	// Verify the keys are deleted.
	assert.Equal(t, 0, len(s.blkRootToPendingAtts), "Did not delete block keys")
}
