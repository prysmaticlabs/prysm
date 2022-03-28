package sync

import (
	"bytes"
	"context"
	"reflect"
	"testing"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsubpb "github.com/libp2p/go-libp2p-pubsub/pb"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/go-bitfield"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/signing"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	lruwrpr "github.com/prysmaticlabs/prysm/cache/lru"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/attestation"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestVerifyIndexInCommittee_CanVerify(t *testing.T) {
	ctx := context.Background()
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MinimalSpecConfig())

	validators := uint64(32)
	s, _ := util.DeterministicGenesisState(t, validators)
	require.NoError(t, s.SetSlot(params.BeaconConfig().SlotsPerEpoch))

	bf := bitfield.NewBitlist(validators / uint64(params.BeaconConfig().SlotsPerEpoch))
	bf.SetBitAt(0, true)
	att := &ethpb.Attestation{Data: &ethpb.AttestationData{
		Target: &ethpb.Checkpoint{Epoch: 0}},
		AggregationBits: bf}

	committee, err := helpers.BeaconCommitteeFromState(context.Background(), s, att.Data.Slot, att.Data.CommitteeIndex)
	assert.NoError(t, err)
	indices, err := attestation.AttestingIndices(att.AggregationBits, committee)
	require.NoError(t, err)
	require.NoError(t, validateIndexInCommittee(ctx, s, att, types.ValidatorIndex(indices[0])))

	wanted := "validator index 1000 is not within the committee"
	assert.ErrorContains(t, wanted, validateIndexInCommittee(ctx, s, att, 1000))
}

func TestVerifyIndexInCommittee_ExistsInBeaconCommittee(t *testing.T) {
	ctx := context.Background()
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MinimalSpecConfig())

	validators := uint64(64)
	s, _ := util.DeterministicGenesisState(t, validators)
	require.NoError(t, s.SetSlot(params.BeaconConfig().SlotsPerEpoch))

	bf := []byte{0xff}
	att := &ethpb.Attestation{Data: &ethpb.AttestationData{
		Target: &ethpb.Checkpoint{Epoch: 0}},
		AggregationBits: bf}

	committee, err := helpers.BeaconCommitteeFromState(context.Background(), s, att.Data.Slot, att.Data.CommitteeIndex)
	require.NoError(t, err)

	require.NoError(t, validateIndexInCommittee(ctx, s, att, committee[0]))

	wanted := "validator index 1000 is not within the committee"
	assert.ErrorContains(t, wanted, validateIndexInCommittee(ctx, s, att, 1000))
}

func TestVerifySelection_NotAnAggregator(t *testing.T) {
	ctx := context.Background()
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MinimalSpecConfig())
	validators := uint64(2048)
	beaconState, privKeys := util.DeterministicGenesisState(t, validators)

	sig := privKeys[0].Sign([]byte{'A'})
	data := util.HydrateAttestationData(&ethpb.AttestationData{})

	_, err := validateSelectionIndex(ctx, beaconState, data, 0, sig.Marshal())
	wanted := "validator is not an aggregator for slot"
	assert.ErrorContains(t, wanted, err)
}

func TestValidateAggregateAndProof_NoBlock(t *testing.T) {
	db := dbtest.SetupDB(t)
	p := p2ptest.NewTestP2P(t)

	att := util.HydrateAttestation(&ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Root: bytesutil.PadTo([]byte("hello-world"), 32)},
			Target: &ethpb.Checkpoint{Root: bytesutil.PadTo([]byte("hello-world"), 32)},
		},
	})

	aggregateAndProof := &ethpb.AggregateAttestationAndProof{
		SelectionProof:  bytesutil.PadTo([]byte{'A'}, fieldparams.BLSSignatureLength),
		Aggregate:       att,
		AggregatorIndex: 0,
	}
	signedAggregateAndProof := &ethpb.SignedAggregateAttestationAndProof{Message: aggregateAndProof, Signature: make([]byte, fieldparams.BLSSignatureLength)}

	c := lruwrpr.New(10)
	r := &Service{
		cfg: &config{
			p2p:         p,
			beaconDB:    db,
			initialSync: &mockSync.Sync{IsSyncing: false},
			attPool:     attestations.NewPool(),
			chain:       &mock.ChainService{},
		},
		blkRootToPendingAtts:           make(map[[32]byte][]*ethpb.SignedAggregateAttestationAndProof),
		seenAggregatedAttestationCache: c,
	}
	r.initCaches()

	buf := new(bytes.Buffer)
	_, err := p.Encoding().EncodeGossip(buf, signedAggregateAndProof)
	require.NoError(t, err)

	topic := p2p.GossipTypeMapping[reflect.TypeOf(signedAggregateAndProof)]
	msg := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data:  buf.Bytes(),
			Topic: &topic,
		},
	}

	if res, err := r.validateAggregateAndProof(context.Background(), "", msg); res == pubsub.ValidationAccept {
		_ = err
		t.Error("Expected validate to fail")
	}
}

func TestValidateAggregateAndProof_NotWithinSlotRange(t *testing.T) {
	db := dbtest.SetupDB(t)
	p := p2ptest.NewTestP2P(t)

	validators := uint64(256)
	beaconState, _ := util.DeterministicGenesisState(t, validators)

	b := util.NewBeaconBlock()
	wsb, err := wrapper.WrappedSignedBeaconBlock(b)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(context.Background(), wsb))
	root, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	s, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, db.SaveState(context.Background(), s, root))

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
		Signature:       make([]byte, fieldparams.BLSSignatureLength),
	}

	aggregateAndProof := &ethpb.AggregateAttestationAndProof{
		Aggregate:      att,
		SelectionProof: make([]byte, fieldparams.BLSSignatureLength),
	}
	signedAggregateAndProof := &ethpb.SignedAggregateAttestationAndProof{Message: aggregateAndProof, Signature: make([]byte, fieldparams.BLSSignatureLength)}

	require.NoError(t, beaconState.SetGenesisTime(uint64(time.Now().Unix())))

	r := &Service{
		cfg: &config{
			p2p:         p,
			beaconDB:    db,
			initialSync: &mockSync.Sync{IsSyncing: false},
			chain: &mock.ChainService{
				Genesis: time.Now(),
				State:   beaconState,
			},
			attPool:             attestations.NewPool(),
			attestationNotifier: (&mock.ChainService{}).OperationNotifier(),
		},
		seenAggregatedAttestationCache: lruwrpr.New(10),
	}
	r.initCaches()

	buf := new(bytes.Buffer)
	_, err = p.Encoding().EncodeGossip(buf, signedAggregateAndProof)
	require.NoError(t, err)

	topic := p2p.GossipTypeMapping[reflect.TypeOf(signedAggregateAndProof)]
	msg := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data:  buf.Bytes(),
			Topic: &topic,
		},
	}

	if res, err := r.validateAggregateAndProof(context.Background(), "", msg); res == pubsub.ValidationAccept {
		_ = err
		t.Error("Expected validate to fail")
	}

	att.Data.Slot = 1<<32 - 1

	buf = new(bytes.Buffer)
	_, err = p.Encoding().EncodeGossip(buf, signedAggregateAndProof)
	require.NoError(t, err)

	msg = &pubsub.Message{
		Message: &pubsubpb.Message{
			Data:  buf.Bytes(),
			Topic: &topic,
		},
	}
	if res, err := r.validateAggregateAndProof(context.Background(), "", msg); res == pubsub.ValidationAccept {
		_ = err
		t.Error("Expected validate to fail")
	}
}

func TestValidateAggregateAndProof_ExistedInPool(t *testing.T) {
	db := dbtest.SetupDB(t)
	p := p2ptest.NewTestP2P(t)

	validators := uint64(256)
	beaconState, _ := util.DeterministicGenesisState(t, validators)

	b := util.NewBeaconBlock()
	wsb, err := wrapper.WrappedSignedBeaconBlock(b)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(context.Background(), wsb))
	root, err := b.Block.HashTreeRoot()
	require.NoError(t, err)

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
		Signature:       make([]byte, fieldparams.BLSSignatureLength),
	}

	aggregateAndProof := &ethpb.AggregateAttestationAndProof{
		Aggregate:      att,
		SelectionProof: make([]byte, fieldparams.BLSSignatureLength),
	}
	signedAggregateAndProof := &ethpb.SignedAggregateAttestationAndProof{Message: aggregateAndProof, Signature: make([]byte, fieldparams.BLSSignatureLength)}

	require.NoError(t, beaconState.SetGenesisTime(uint64(time.Now().Unix())))
	r := &Service{
		cfg: &config{
			attPool:     attestations.NewPool(),
			p2p:         p,
			beaconDB:    db,
			initialSync: &mockSync.Sync{IsSyncing: false},
			chain: &mock.ChainService{Genesis: time.Now(),
				State: beaconState},
			attestationNotifier: (&mock.ChainService{}).OperationNotifier(),
		},
		seenAggregatedAttestationCache: lruwrpr.New(10),
		blkRootToPendingAtts:           make(map[[32]byte][]*ethpb.SignedAggregateAttestationAndProof),
	}
	r.initCaches()

	buf := new(bytes.Buffer)
	_, err = p.Encoding().EncodeGossip(buf, signedAggregateAndProof)
	require.NoError(t, err)

	topic := p2p.GossipTypeMapping[reflect.TypeOf(signedAggregateAndProof)]
	msg := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data:  buf.Bytes(),
			Topic: &topic,
		},
	}

	require.NoError(t, r.cfg.attPool.SaveBlockAttestation(att))
	if res, err := r.validateAggregateAndProof(context.Background(), "", msg); res == pubsub.ValidationAccept {
		_ = err
		t.Error("Expected validate to fail")
	}
}

func TestValidateAggregateAndProof_CanValidate(t *testing.T) {
	db := dbtest.SetupDB(t)
	p := p2ptest.NewTestP2P(t)

	validators := uint64(256)
	beaconState, privKeys := util.DeterministicGenesisState(t, validators)

	b := util.NewBeaconBlock()
	wsb, err := wrapper.WrappedSignedBeaconBlock(b)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(context.Background(), wsb))
	root, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	s, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, db.SaveState(context.Background(), s, root))

	aggBits := bitfield.NewBitlist(validators / uint64(params.BeaconConfig().SlotsPerEpoch))
	aggBits.SetBitAt(0, true)
	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Slot:            1,
			BeaconBlockRoot: root[:],
			Source:          &ethpb.Checkpoint{Epoch: 0, Root: bytesutil.PadTo([]byte("hello-world"), 32)},
			Target:          &ethpb.Checkpoint{Epoch: 0, Root: root[:]},
		},
		AggregationBits: aggBits,
	}

	committee, err := helpers.BeaconCommitteeFromState(context.Background(), beaconState, att.Data.Slot, att.Data.CommitteeIndex)
	assert.NoError(t, err)
	attestingIndices, err := attestation.AttestingIndices(att.AggregationBits, committee)
	require.NoError(t, err)
	assert.NoError(t, err)
	attesterDomain, err := signing.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainBeaconAttester, beaconState.GenesisValidatorsRoot())
	assert.NoError(t, err)
	hashTreeRoot, err := signing.ComputeSigningRoot(att.Data, attesterDomain)
	assert.NoError(t, err)
	sigs := make([]bls.Signature, len(attestingIndices))
	for i, indice := range attestingIndices {
		sig := privKeys[indice].Sign(hashTreeRoot[:])
		sigs[i] = sig
	}
	att.Signature = bls.AggregateSignatures(sigs).Marshal()
	ai := committee[0]
	sszUint := types.SSZUint64(att.Data.Slot)
	sig, err := signing.ComputeDomainAndSign(beaconState, 0, &sszUint, params.BeaconConfig().DomainSelectionProof, privKeys[ai])
	require.NoError(t, err)
	aggregateAndProof := &ethpb.AggregateAttestationAndProof{
		SelectionProof:  sig,
		Aggregate:       att,
		AggregatorIndex: ai,
	}
	signedAggregateAndProof := &ethpb.SignedAggregateAttestationAndProof{Message: aggregateAndProof}
	signedAggregateAndProof.Signature, err = signing.ComputeDomainAndSign(beaconState, 0, signedAggregateAndProof.Message, params.BeaconConfig().DomainAggregateAndProof, privKeys[ai])
	require.NoError(t, err)

	require.NoError(t, beaconState.SetGenesisTime(uint64(time.Now().Unix())))
	r := &Service{
		cfg: &config{
			p2p:         p,
			beaconDB:    db,
			initialSync: &mockSync.Sync{IsSyncing: false},
			chain: &mock.ChainService{Genesis: time.Now().Add(-oneEpoch()),
				State:            beaconState,
				ValidAttestation: true,
				FinalizedCheckPoint: &ethpb.Checkpoint{
					Epoch: 0,
					Root:  att.Data.BeaconBlockRoot,
				}},
			attPool:             attestations.NewPool(),
			attestationNotifier: (&mock.ChainService{}).OperationNotifier(),
		},
		seenAggregatedAttestationCache: lruwrpr.New(10),
	}
	r.initCaches()

	buf := new(bytes.Buffer)
	_, err = p.Encoding().EncodeGossip(buf, signedAggregateAndProof)
	require.NoError(t, err)

	topic := p2p.GossipTypeMapping[reflect.TypeOf(signedAggregateAndProof)]
	d, err := r.currentForkDigest()
	assert.NoError(t, err)
	topic = r.addDigestToTopic(topic, d)
	msg := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data:  buf.Bytes(),
			Topic: &topic,
		},
	}
	res, err := r.validateAggregateAndProof(context.Background(), "", msg)
	assert.NoError(t, err)
	assert.Equal(t, pubsub.ValidationAccept, res, "Validated status is false")
	assert.NotNil(t, msg.ValidatorData, "Did not set validator data")
}

func TestVerifyIndexInCommittee_SeenAggregatorEpoch(t *testing.T) {
	db := dbtest.SetupDB(t)
	p := p2ptest.NewTestP2P(t)

	validators := uint64(256)
	beaconState, privKeys := util.DeterministicGenesisState(t, validators)

	b := util.NewBeaconBlock()
	wsb, err := wrapper.WrappedSignedBeaconBlock(b)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(context.Background(), wsb))
	root, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	s, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, db.SaveState(context.Background(), s, root))

	aggBits := bitfield.NewBitlist(validators / uint64(params.BeaconConfig().SlotsPerEpoch))
	aggBits.SetBitAt(0, true)
	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Slot:            1,
			BeaconBlockRoot: root[:],
			Source:          &ethpb.Checkpoint{Epoch: 0, Root: bytesutil.PadTo([]byte("hello-world"), 32)},
			Target:          &ethpb.Checkpoint{Epoch: 0, Root: root[:]},
		},
		AggregationBits: aggBits,
	}

	committee, err := helpers.BeaconCommitteeFromState(context.Background(), beaconState, att.Data.Slot, att.Data.CommitteeIndex)
	require.NoError(t, err)
	attestingIndices, err := attestation.AttestingIndices(att.AggregationBits, committee)
	require.NoError(t, err)
	attesterDomain, err := signing.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainBeaconAttester, beaconState.GenesisValidatorsRoot())
	require.NoError(t, err)
	hashTreeRoot, err := signing.ComputeSigningRoot(att.Data, attesterDomain)
	assert.NoError(t, err)
	sigs := make([]bls.Signature, len(attestingIndices))
	for i, indice := range attestingIndices {
		sig := privKeys[indice].Sign(hashTreeRoot[:])
		sigs[i] = sig
	}
	att.Signature = bls.AggregateSignatures(sigs).Marshal()
	ai := committee[0]
	sszUint := types.SSZUint64(att.Data.Slot)
	sig, err := signing.ComputeDomainAndSign(beaconState, 0, &sszUint, params.BeaconConfig().DomainSelectionProof, privKeys[ai])
	require.NoError(t, err)
	aggregateAndProof := &ethpb.AggregateAttestationAndProof{
		SelectionProof:  sig,
		Aggregate:       att,
		AggregatorIndex: ai,
	}
	signedAggregateAndProof := &ethpb.SignedAggregateAttestationAndProof{Message: aggregateAndProof}
	signedAggregateAndProof.Signature, err = signing.ComputeDomainAndSign(beaconState, 0, signedAggregateAndProof.Message, params.BeaconConfig().DomainAggregateAndProof, privKeys[ai])
	require.NoError(t, err)
	require.NoError(t, beaconState.SetGenesisTime(uint64(time.Now().Unix())))

	r := &Service{
		cfg: &config{
			p2p:         p,
			beaconDB:    db,
			initialSync: &mockSync.Sync{IsSyncing: false},
			chain: &mock.ChainService{Genesis: time.Now().Add(-oneEpoch()),
				ValidatorsRoot:   [32]byte{'A'},
				State:            beaconState,
				ValidAttestation: true,
				FinalizedCheckPoint: &ethpb.Checkpoint{
					Epoch: 0,
					Root:  signedAggregateAndProof.Message.Aggregate.Data.BeaconBlockRoot,
				}},

			attPool:             attestations.NewPool(),
			attestationNotifier: (&mock.ChainService{}).OperationNotifier(),
		},
		seenAggregatedAttestationCache: lruwrpr.New(10),
	}
	r.initCaches()

	buf := new(bytes.Buffer)
	_, err = p.Encoding().EncodeGossip(buf, signedAggregateAndProof)
	require.NoError(t, err)

	topic := p2p.GossipTypeMapping[reflect.TypeOf(signedAggregateAndProof)]
	d, err := r.currentForkDigest()
	assert.NoError(t, err)
	topic = r.addDigestToTopic(topic, d)
	msg := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data:  buf.Bytes(),
			Topic: &topic,
		},
	}
	res, err := r.validateAggregateAndProof(context.Background(), "", msg)
	assert.NoError(t, err)
	require.Equal(t, pubsub.ValidationAccept, res, "Validated status is false")

	// Should fail with another attestation in the same epoch.
	signedAggregateAndProof.Message.Aggregate.Data.Slot++
	buf = new(bytes.Buffer)
	_, err = p.Encoding().EncodeGossip(buf, signedAggregateAndProof)
	require.NoError(t, err)
	msg = &pubsub.Message{
		Message: &pubsubpb.Message{
			Data:  buf.Bytes(),
			Topic: &topic,
		},
	}

	time.Sleep(10 * time.Millisecond) // Wait for cached value to pass through buffers.
	if res, err := r.validateAggregateAndProof(context.Background(), "", msg); res == pubsub.ValidationAccept {
		_ = err
		t.Fatal("Validated status is true")
	}
}

func TestValidateAggregateAndProof_BadBlock(t *testing.T) {

	db := dbtest.SetupDB(t)
	p := p2ptest.NewTestP2P(t)

	validators := uint64(256)
	beaconState, privKeys := util.DeterministicGenesisState(t, validators)

	b := util.NewBeaconBlock()
	root, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	s, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, db.SaveState(context.Background(), s, root))

	aggBits := bitfield.NewBitlist(validators / uint64(params.BeaconConfig().SlotsPerEpoch))
	aggBits.SetBitAt(0, true)
	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			BeaconBlockRoot: root[:],
			Source:          &ethpb.Checkpoint{Epoch: 0, Root: bytesutil.PadTo([]byte("hello-world"), 32)},
			Target:          &ethpb.Checkpoint{Epoch: 0, Root: root[:]},
		},
		AggregationBits: aggBits,
	}

	committee, err := helpers.BeaconCommitteeFromState(context.Background(), beaconState, att.Data.Slot, att.Data.CommitteeIndex)
	assert.NoError(t, err)
	attestingIndices, err := attestation.AttestingIndices(att.AggregationBits, committee)
	require.NoError(t, err)
	assert.NoError(t, err)
	attesterDomain, err := signing.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainBeaconAttester, beaconState.GenesisValidatorsRoot())
	assert.NoError(t, err)
	hashTreeRoot, err := signing.ComputeSigningRoot(att.Data, attesterDomain)
	assert.NoError(t, err)
	sigs := make([]bls.Signature, len(attestingIndices))
	for i, indice := range attestingIndices {
		sig := privKeys[indice].Sign(hashTreeRoot[:])
		sigs[i] = sig
	}
	att.Signature = bls.AggregateSignatures(sigs).Marshal()
	ai := committee[0]
	sszUint := types.SSZUint64(att.Data.Slot)
	sig, err := signing.ComputeDomainAndSign(beaconState, 0, &sszUint, params.BeaconConfig().DomainSelectionProof, privKeys[ai])
	require.NoError(t, err)

	aggregateAndProof := &ethpb.AggregateAttestationAndProof{
		SelectionProof:  sig,
		Aggregate:       att,
		AggregatorIndex: ai,
	}
	signedAggregateAndProof := &ethpb.SignedAggregateAttestationAndProof{Message: aggregateAndProof}
	signedAggregateAndProof.Signature, err = signing.ComputeDomainAndSign(beaconState, 0, signedAggregateAndProof.Message, params.BeaconConfig().DomainAggregateAndProof, privKeys[ai])
	require.NoError(t, err)

	require.NoError(t, beaconState.SetGenesisTime(uint64(time.Now().Unix())))
	r := &Service{
		cfg: &config{
			p2p:         p,
			beaconDB:    db,
			initialSync: &mockSync.Sync{IsSyncing: false},
			chain: &mock.ChainService{Genesis: time.Now(),
				State:            beaconState,
				ValidAttestation: true,
				FinalizedCheckPoint: &ethpb.Checkpoint{
					Epoch: 0,
				}},
			attPool:             attestations.NewPool(),
			attestationNotifier: (&mock.ChainService{}).OperationNotifier(),
		},
		seenAggregatedAttestationCache: lruwrpr.New(10),
	}
	r.initCaches()
	// Set beacon block as bad.
	r.setBadBlock(context.Background(), root)
	buf := new(bytes.Buffer)
	_, err = p.Encoding().EncodeGossip(buf, signedAggregateAndProof)
	require.NoError(t, err)

	topic := p2p.GossipTypeMapping[reflect.TypeOf(signedAggregateAndProof)]
	msg := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data:  buf.Bytes(),
			Topic: &topic,
		},
	}
	res, err := r.validateAggregateAndProof(context.Background(), "", msg)
	assert.NotNil(t, err)
	assert.Equal(t, pubsub.ValidationReject, res, "Validated status is true")
}

func TestValidateAggregateAndProof_RejectWhenAttEpochDoesntEqualTargetEpoch(t *testing.T) {
	db := dbtest.SetupDB(t)
	p := p2ptest.NewTestP2P(t)

	validators := uint64(256)
	beaconState, privKeys := util.DeterministicGenesisState(t, validators)

	b := util.NewBeaconBlock()
	wsb, err := wrapper.WrappedSignedBeaconBlock(b)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(context.Background(), wsb))
	root, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	s, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, db.SaveState(context.Background(), s, root))

	aggBits := bitfield.NewBitlist(validators / uint64(params.BeaconConfig().SlotsPerEpoch))
	aggBits.SetBitAt(0, true)
	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			BeaconBlockRoot: root[:],
			Source:          &ethpb.Checkpoint{Epoch: 0, Root: bytesutil.PadTo([]byte("hello-world"), 32)},
			Target:          &ethpb.Checkpoint{Epoch: 1, Root: root[:]},
		},
		AggregationBits: aggBits,
	}

	committee, err := helpers.BeaconCommitteeFromState(context.Background(), beaconState, att.Data.Slot, att.Data.CommitteeIndex)
	assert.NoError(t, err)
	attestingIndices, err := attestation.AttestingIndices(att.AggregationBits, committee)
	require.NoError(t, err)
	assert.NoError(t, err)
	attesterDomain, err := signing.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainBeaconAttester, beaconState.GenesisValidatorsRoot())
	assert.NoError(t, err)
	hashTreeRoot, err := signing.ComputeSigningRoot(att.Data, attesterDomain)
	assert.NoError(t, err)
	sigs := make([]bls.Signature, len(attestingIndices))
	for i, indice := range attestingIndices {
		sig := privKeys[indice].Sign(hashTreeRoot[:])
		sigs[i] = sig
	}
	att.Signature = bls.AggregateSignatures(sigs).Marshal()
	ai := committee[0]
	sszUint := types.SSZUint64(att.Data.Slot)
	sig, err := signing.ComputeDomainAndSign(beaconState, 0, &sszUint, params.BeaconConfig().DomainSelectionProof, privKeys[ai])
	require.NoError(t, err)
	aggregateAndProof := &ethpb.AggregateAttestationAndProof{
		SelectionProof:  sig,
		Aggregate:       att,
		AggregatorIndex: ai,
	}
	signedAggregateAndProof := &ethpb.SignedAggregateAttestationAndProof{Message: aggregateAndProof}
	signedAggregateAndProof.Signature, err = signing.ComputeDomainAndSign(beaconState, 0, signedAggregateAndProof.Message, params.BeaconConfig().DomainAggregateAndProof, privKeys[ai])
	require.NoError(t, err)

	require.NoError(t, beaconState.SetGenesisTime(uint64(time.Now().Unix())))
	r := &Service{
		cfg: &config{
			p2p:         p,
			beaconDB:    db,
			initialSync: &mockSync.Sync{IsSyncing: false},
			chain: &mock.ChainService{Genesis: time.Now(),
				State:            beaconState,
				ValidAttestation: true,
				FinalizedCheckPoint: &ethpb.Checkpoint{
					Epoch: 0,
					Root:  att.Data.BeaconBlockRoot,
				}},
			attPool:             attestations.NewPool(),
			attestationNotifier: (&mock.ChainService{}).OperationNotifier(),
		},
		seenAggregatedAttestationCache: lruwrpr.New(10),
	}
	r.initCaches()

	buf := new(bytes.Buffer)
	_, err = p.Encoding().EncodeGossip(buf, signedAggregateAndProof)
	require.NoError(t, err)

	topic := p2p.GossipTypeMapping[reflect.TypeOf(signedAggregateAndProof)]
	msg := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data:  buf.Bytes(),
			Topic: &topic,
		},
	}
	res, err := r.validateAggregateAndProof(context.Background(), "", msg)
	assert.NotNil(t, err)
	assert.Equal(t, pubsub.ValidationReject, res)
}

func TestValidateAggregateAndProof_Optimistic(t *testing.T) {
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()

	exit, s := setupValidExit(t)

	r := &Service{
		cfg: &config{
			p2p: p,
			chain: &mock.ChainService{
				State:      s,
				Optimistic: true,
			},
			initialSync: &mockSync.Sync{IsSyncing: false},
		},
	}
	buf := new(bytes.Buffer)
	_, err := p.Encoding().EncodeGossip(buf, exit)
	require.NoError(t, err)
	topic := p2p.GossipTypeMapping[reflect.TypeOf(exit)]
	m := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data:  buf.Bytes(),
			Topic: &topic,
		},
	}
	res, err := r.validateAggregateAndProof(ctx, "", m)
	assert.NoError(t, err)
	valid := res == pubsub.ValidationIgnore
	assert.Equal(t, true, valid, "Validation should have ignored the message")
}
