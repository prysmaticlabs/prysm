package sync

import (
	"bytes"
	"context"
	"math/rand"
	"reflect"
	"testing"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsubpb "github.com/libp2p/go-libp2p-pubsub/pb"
	types "github.com/prysmaticlabs/eth2-types"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func setupValidAttesterSlashing(t *testing.T) (*ethpb.AttesterSlashing, state.BeaconState) {
	state, privKeys := util.DeterministicGenesisState(t, 5)
	vals := state.Validators()
	for _, vv := range vals {
		vv.WithdrawableEpoch = types.Epoch(1 * params.BeaconConfig().SlotsPerEpoch)
	}
	require.NoError(t, state.SetValidators(vals))

	att1 := util.HydrateIndexedAttestation(&ethpb.IndexedAttestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 1},
		},
		AttestingIndices: []uint64{0, 1},
	})
	domain, err := signing.Domain(state.Fork(), 0, params.BeaconConfig().DomainBeaconAttester, state.GenesisValidatorsRoot())
	require.NoError(t, err)
	hashTreeRoot, err := signing.ComputeSigningRoot(att1.Data, domain)
	assert.NoError(t, err)
	sig0 := privKeys[0].Sign(hashTreeRoot[:])
	sig1 := privKeys[1].Sign(hashTreeRoot[:])
	aggregateSig := bls.AggregateSignatures([]bls.Signature{sig0, sig1})
	att1.Signature = aggregateSig.Marshal()

	att2 := util.HydrateIndexedAttestation(&ethpb.IndexedAttestation{
		AttestingIndices: []uint64{0, 1},
	})
	hashTreeRoot, err = signing.ComputeSigningRoot(att2.Data, domain)
	assert.NoError(t, err)
	sig0 = privKeys[0].Sign(hashTreeRoot[:])
	sig1 = privKeys[1].Sign(hashTreeRoot[:])
	aggregateSig = bls.AggregateSignatures([]bls.Signature{sig0, sig1})
	att2.Signature = aggregateSig.Marshal()

	slashing := &ethpb.AttesterSlashing{
		Attestation_1: att1,
		Attestation_2: att2,
	}

	currentSlot := 2 * params.BeaconConfig().SlotsPerEpoch
	require.NoError(t, state.SetSlot(currentSlot))

	b := make([]byte, 32)
	_, err = rand.Read(b)
	require.NoError(t, err)

	return slashing, state
}

func TestValidateAttesterSlashing_ValidSlashing(t *testing.T) {
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()

	slashing, s := setupValidAttesterSlashing(t)

	r := &Service{
		cfg: &config{
			p2p:         p,
			chain:       &mock.ChainService{State: s, Genesis: time.Now()},
			initialSync: &mockSync.Sync{IsSyncing: false},
		},
		seenAttesterSlashingCache: make(map[uint64]bool),
		subHandler:                newSubTopicHandler(),
	}

	buf := new(bytes.Buffer)
	_, err := p.Encoding().EncodeGossip(buf, slashing)
	require.NoError(t, err)

	topic := p2p.GossipTypeMapping[reflect.TypeOf(slashing)]
	d, err := r.currentForkDigest()
	assert.NoError(t, err)
	topic = r.addDigestToTopic(topic, d)
	msg := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data:  buf.Bytes(),
			Topic: &topic,
		},
	}
	res, err := r.validateAttesterSlashing(ctx, "foobar", msg)
	assert.NoError(t, err)
	valid := res == pubsub.ValidationAccept

	assert.Equal(t, true, valid, "Failed Validation")
	assert.NotNil(t, msg.ValidatorData, "Decoded message was not set on the message validator data")
}

func TestValidateAttesterSlashing_CanFilter(t *testing.T) {
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()

	r := &Service{
		cfg: &config{
			p2p:         p,
			initialSync: &mockSync.Sync{IsSyncing: false},
			chain:       &mock.ChainService{Genesis: time.Now()},
		},
		seenAttesterSlashingCache: make(map[uint64]bool),
		subHandler:                newSubTopicHandler(),
	}

	r.setAttesterSlashingIndicesSeen([]uint64{1, 2, 3, 4}, []uint64{3, 4, 5, 6})

	// The below attestations should be filtered hence bad signature is ok.
	topic := p2p.GossipTypeMapping[reflect.TypeOf(&ethpb.AttesterSlashing{})]
	d, err := r.currentForkDigest()
	assert.NoError(t, err)
	topic = r.addDigestToTopic(topic, d)
	buf := new(bytes.Buffer)
	_, err = p.Encoding().EncodeGossip(buf, &ethpb.AttesterSlashing{
		Attestation_1: util.HydrateIndexedAttestation(&ethpb.IndexedAttestation{
			AttestingIndices: []uint64{3},
		}),
		Attestation_2: util.HydrateIndexedAttestation(&ethpb.IndexedAttestation{
			AttestingIndices: []uint64{3},
		}),
	})
	require.NoError(t, err)
	msg := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data:  buf.Bytes(),
			Topic: &topic,
		},
	}
	res, err := r.validateAttesterSlashing(ctx, "foobar", msg)
	_ = err
	ignored := res == pubsub.ValidationIgnore
	assert.Equal(t, true, ignored)

	buf = new(bytes.Buffer)
	_, err = p.Encoding().EncodeGossip(buf, &ethpb.AttesterSlashing{
		Attestation_1: util.HydrateIndexedAttestation(&ethpb.IndexedAttestation{
			AttestingIndices: []uint64{4, 3},
		}),
		Attestation_2: util.HydrateIndexedAttestation(&ethpb.IndexedAttestation{
			AttestingIndices: []uint64{3, 4},
		}),
	})
	require.NoError(t, err)
	msg = &pubsub.Message{
		Message: &pubsubpb.Message{
			Data:  buf.Bytes(),
			Topic: &topic,
		},
	}
	res, err = r.validateAttesterSlashing(ctx, "foobar", msg)
	_ = err
	ignored = res == pubsub.ValidationIgnore
	assert.Equal(t, true, ignored)
}

func TestValidateAttesterSlashing_ContextTimeout(t *testing.T) {
	p := p2ptest.NewTestP2P(t)

	slashing, state := setupValidAttesterSlashing(t)
	slashing.Attestation_1.Data.Target.Epoch = 100000000

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	r := &Service{
		cfg: &config{
			p2p:         p,
			chain:       &mock.ChainService{State: state},
			initialSync: &mockSync.Sync{IsSyncing: false},
		},
		seenAttesterSlashingCache: make(map[uint64]bool),
	}

	buf := new(bytes.Buffer)
	_, err := p.Encoding().EncodeGossip(buf, slashing)
	require.NoError(t, err)

	topic := p2p.GossipTypeMapping[reflect.TypeOf(slashing)]
	msg := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data:  buf.Bytes(),
			Topic: &topic,
		},
	}
	res, err := r.validateAttesterSlashing(ctx, "foobar", msg)
	_ = err
	valid := res == pubsub.ValidationAccept
	assert.Equal(t, false, valid, "slashing from the far distant future should have timed out and returned false")
}

func TestValidateAttesterSlashing_Syncing(t *testing.T) {
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()

	slashing, s := setupValidAttesterSlashing(t)

	r := &Service{
		cfg: &config{
			p2p:         p,
			chain:       &mock.ChainService{State: s},
			initialSync: &mockSync.Sync{IsSyncing: true},
		},
	}

	buf := new(bytes.Buffer)
	_, err := p.Encoding().EncodeGossip(buf, slashing)
	require.NoError(t, err)

	topic := p2p.GossipTypeMapping[reflect.TypeOf(slashing)]
	msg := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data:  buf.Bytes(),
			Topic: &topic,
		},
	}
	res, err := r.validateAttesterSlashing(ctx, "foobar", msg)
	_ = err
	valid := res == pubsub.ValidationAccept
	assert.Equal(t, false, valid, "Passed validation")
}

func TestSeenAttesterSlashingIndices(t *testing.T) {
	tt := []struct {
		saveIndices1  []uint64
		saveIndices2  []uint64
		checkIndices1 []uint64
		checkIndices2 []uint64
		seen          bool
	}{
		{
			saveIndices1:  []uint64{0, 1, 2},
			saveIndices2:  []uint64{0},
			checkIndices1: []uint64{0, 1, 2},
			checkIndices2: []uint64{0},
			seen:          true,
		},
		{
			saveIndices1:  []uint64{100, 99, 98},
			saveIndices2:  []uint64{99, 98, 97},
			checkIndices1: []uint64{99, 98},
			checkIndices2: []uint64{99, 98},
			seen:          true,
		},
		{
			saveIndices1:  []uint64{100},
			saveIndices2:  []uint64{100},
			checkIndices1: []uint64{100, 101},
			checkIndices2: []uint64{100, 101},
			seen:          false,
		},
		{
			saveIndices1:  []uint64{100, 99, 98},
			saveIndices2:  []uint64{99, 98, 97},
			checkIndices1: []uint64{99, 98, 97},
			checkIndices2: []uint64{99, 98, 97},
			seen:          false,
		},
		{
			saveIndices1:  []uint64{100, 99, 98},
			saveIndices2:  []uint64{99, 98, 97},
			checkIndices1: []uint64{101, 100},
			checkIndices2: []uint64{101},
			seen:          false,
		},
	}
	for _, tc := range tt {
		r := &Service{
			seenAttesterSlashingCache: map[uint64]bool{},
		}
		r.setAttesterSlashingIndicesSeen(tc.saveIndices1, tc.saveIndices2)
		assert.Equal(t, tc.seen, r.hasSeenAttesterSlashingIndices(tc.checkIndices1, tc.checkIndices2))
	}
}

func TestValidateAttesterSlashing_Optimistic(t *testing.T) {
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()

	slashing, s := setupValidAttesterSlashing(t)

	r := &Service{
		cfg: &config{
			p2p:         p,
			chain:       &mock.ChainService{State: s, Optimistic: true},
			initialSync: &mockSync.Sync{IsSyncing: false},
		},
	}

	buf := new(bytes.Buffer)
	_, err := p.Encoding().EncodeGossip(buf, slashing)
	require.NoError(t, err)

	topic := p2p.GossipTypeMapping[reflect.TypeOf(slashing)]
	msg := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data:  buf.Bytes(),
			Topic: &topic,
		},
	}
	res, err := r.validateAttesterSlashing(ctx, "foobar", msg)
	assert.NoError(t, err)
	valid := res == pubsub.ValidationIgnore
	assert.Equal(t, true, valid, "Should have ignore this message")
}
