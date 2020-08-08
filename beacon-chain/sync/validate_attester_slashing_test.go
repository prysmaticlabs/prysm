package sync

import (
	"bytes"
	"context"
	"math/rand"
	"reflect"
	"testing"
	"time"

	lru "github.com/hashicorp/golang-lru"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsubpb "github.com/libp2p/go-libp2p-pubsub/pb"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func setupValidAttesterSlashing(t *testing.T) (*ethpb.AttesterSlashing, *stateTrie.BeaconState) {
	state, privKeys := testutil.DeterministicGenesisState(t, 5)
	vals := state.Validators()
	for _, vv := range vals {
		vv.WithdrawableEpoch = 1 * params.BeaconConfig().SlotsPerEpoch
	}
	require.NoError(t, state.SetValidators(vals))

	att1 := &ethpb.IndexedAttestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
			Target: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
		},
		AttestingIndices: []uint64{0, 1},
	}
	domain, err := helpers.Domain(state.Fork(), 0, params.BeaconConfig().DomainBeaconAttester, state.GenesisValidatorRoot())
	require.NoError(t, err)
	hashTreeRoot, err := helpers.ComputeSigningRoot(att1.Data, domain)
	assert.NoError(t, err)
	sig0 := privKeys[0].Sign(hashTreeRoot[:])
	sig1 := privKeys[1].Sign(hashTreeRoot[:])
	aggregateSig := bls.AggregateSignatures([]bls.Signature{sig0, sig1})
	att1.Signature = aggregateSig.Marshal()[:]

	att2 := &ethpb.IndexedAttestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
			Target: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
		},
		AttestingIndices: []uint64{0, 1},
	}
	hashTreeRoot, err = helpers.ComputeSigningRoot(att2.Data, domain)
	assert.NoError(t, err)
	sig0 = privKeys[0].Sign(hashTreeRoot[:])
	sig1 = privKeys[1].Sign(hashTreeRoot[:])
	aggregateSig = bls.AggregateSignatures([]bls.Signature{sig0, sig1})
	att2.Signature = aggregateSig.Marshal()[:]

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

	c, err := lru.New(10)
	require.NoError(t, err)
	r := &Service{
		p2p:                       p,
		chain:                     &mock.ChainService{State: s},
		initialSync:               &mockSync.Sync{IsSyncing: false},
		seenAttesterSlashingCache: c,
	}

	buf := new(bytes.Buffer)
	_, err = p.Encoding().EncodeGossip(buf, slashing)
	require.NoError(t, err)

	msg := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(slashing)],
			},
		},
	}
	valid := r.validateAttesterSlashing(ctx, "foobar", msg) == pubsub.ValidationAccept

	assert.Equal(t, true, valid, "Failed Validation")
	assert.NotNil(t, msg.ValidatorData, "Decoded message was not set on the message validator data")
}

func TestValidateAttesterSlashing_ContextTimeout(t *testing.T) {
	p := p2ptest.NewTestP2P(t)

	slashing, state := setupValidAttesterSlashing(t)
	slashing.Attestation_1.Data.Target.Epoch = 100000000

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	c, err := lru.New(10)
	require.NoError(t, err)
	r := &Service{
		p2p:                       p,
		chain:                     &mock.ChainService{State: state},
		initialSync:               &mockSync.Sync{IsSyncing: false},
		seenAttesterSlashingCache: c,
	}

	buf := new(bytes.Buffer)
	_, err = p.Encoding().EncodeGossip(buf, slashing)
	require.NoError(t, err)

	msg := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(slashing)],
			},
		},
	}
	valid := r.validateAttesterSlashing(ctx, "", msg) == pubsub.ValidationAccept
	assert.Equal(t, false, valid, "slashing from the far distant future should have timed out and returned false")
}

func TestValidateAttesterSlashing_Syncing(t *testing.T) {
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()

	slashing, s := setupValidAttesterSlashing(t)

	r := &Service{
		p2p:         p,
		chain:       &mock.ChainService{State: s},
		initialSync: &mockSync.Sync{IsSyncing: true},
	}

	buf := new(bytes.Buffer)
	_, err := p.Encoding().EncodeGossip(buf, slashing)
	require.NoError(t, err)
	msg := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(slashing)],
			},
		},
	}
	valid := r.validateAttesterSlashing(ctx, "", msg) == pubsub.ValidationAccept
	assert.Equal(t, false, valid, "Passed validation")
}
