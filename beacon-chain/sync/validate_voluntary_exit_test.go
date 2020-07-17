package sync

import (
	"bytes"
	"context"
	"crypto/rand"
	"reflect"
	"testing"

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
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func setupValidExit(t *testing.T) (*ethpb.SignedVoluntaryExit, *stateTrie.BeaconState) {
	exit := &ethpb.SignedVoluntaryExit{
		Exit: &ethpb.VoluntaryExit{
			ValidatorIndex: 0,
			Epoch:          1 + params.BeaconConfig().ShardCommitteePeriod,
		},
	}
	registry := []*ethpb.Validator{
		{
			ExitEpoch:       params.BeaconConfig().FarFutureEpoch,
			ActivationEpoch: 0,
		},
	}
	state, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
		Fork: &pb.Fork{
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
		},
		Slot: params.BeaconConfig().SlotsPerEpoch * 5,
	})
	require.NoError(t, err)
	err = state.SetSlot(state.Slot() + (params.BeaconConfig().ShardCommitteePeriod * params.BeaconConfig().SlotsPerEpoch))
	require.NoError(t, err)
	domain, err := helpers.Domain(state.Fork(), helpers.CurrentEpoch(state), params.BeaconConfig().DomainVoluntaryExit, state.GenesisValidatorRoot())
	require.NoError(t, err)
	signingRoot, err := helpers.ComputeSigningRoot(exit.Exit, domain)
	require.NoError(t, err)
	priv := bls.RandKey()

	sig := priv.Sign(signingRoot[:])
	exit.Signature = sig.Marshal()

	val, err := state.ValidatorAtIndex(0)
	require.NoError(t, err)
	val.PublicKey = priv.PublicKey().Marshal()[:]
	require.NoError(t, state.UpdateValidatorAtIndex(0, val))

	b := make([]byte, 32)
	_, err = rand.Read(b)
	require.NoError(t, err)

	return exit, state
}

func TestValidateVoluntaryExit_ValidExit(t *testing.T) {
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()

	exit, s := setupValidExit(t)

	c, err := lru.New(10)
	require.NoError(t, err)
	r := &Service{
		p2p: p,
		chain: &mock.ChainService{
			State: s,
		},
		initialSync:   &mockSync.Sync{IsSyncing: false},
		seenExitCache: c,
	}

	buf := new(bytes.Buffer)
	_, err = p.Encoding().EncodeGossip(buf, exit)
	require.NoError(t, err)
	m := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(exit)],
			},
		},
	}
	valid := r.validateVoluntaryExit(ctx, "", m) == pubsub.ValidationAccept
	assert.Equal(t, true, valid, "Failed validation")
	assert.NotNil(t, m.ValidatorData, "Decoded message was not set on the message validator data")
}

func TestValidateVoluntaryExit_ValidExit_Syncing(t *testing.T) {
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()

	exit, s := setupValidExit(t)

	r := &Service{
		p2p: p,
		chain: &mock.ChainService{
			State: s,
		},
		initialSync: &mockSync.Sync{IsSyncing: true},
	}
	buf := new(bytes.Buffer)
	_, err := p.Encoding().EncodeGossip(buf, exit)
	require.NoError(t, err)
	m := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(exit)],
			},
		},
	}
	valid := r.validateVoluntaryExit(ctx, "", m) == pubsub.ValidationAccept
	assert.Equal(t, false, valid, "Validation should have failed")
}
