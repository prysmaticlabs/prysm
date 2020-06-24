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
	if err != nil {
		t.Fatal(err)
	}
	if err := state.SetSlot(
		state.Slot() + (params.BeaconConfig().ShardCommitteePeriod * params.BeaconConfig().SlotsPerEpoch),
	); err != nil {
		t.Fatal(err)
	}
	domain, err := helpers.Domain(state.Fork(), helpers.CurrentEpoch(state), params.BeaconConfig().DomainVoluntaryExit, state.GenesisValidatorRoot())
	if err != nil {
		t.Fatal(err)
	}
	signingRoot, err := helpers.ComputeSigningRoot(exit.Exit, domain)
	if err != nil {
		t.Error(err)
	}
	priv := bls.RandKey()

	sig := priv.Sign(signingRoot[:])
	exit.Signature = sig.Marshal()

	val, err := state.ValidatorAtIndex(0)
	if err != nil {
		t.Fatal(err)
	}
	val.PublicKey = priv.PublicKey().Marshal()[:]
	if err := state.UpdateValidatorAtIndex(0, val); err != nil {
		t.Fatal(err)
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}

	return exit, state
}

func TestValidateVoluntaryExit_ValidExit(t *testing.T) {
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()

	exit, s := setupValidExit(t)

	c, err := lru.New(10)
	if err != nil {
		t.Fatal(err)
	}
	r := &Service{
		p2p: p,
		chain: &mock.ChainService{
			State: s,
		},
		initialSync:   &mockSync.Sync{IsSyncing: false},
		seenExitCache: c,
	}

	buf := new(bytes.Buffer)
	if _, err := p.Encoding().EncodeGossip(buf, exit); err != nil {
		t.Fatal(err)
	}
	m := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(exit)],
			},
		},
	}
	valid := r.validateVoluntaryExit(ctx, "", m) == pubsub.ValidationAccept
	if !valid {
		t.Error("Failed validation")
	}

	if m.ValidatorData == nil {
		t.Error("Decoded message was not set on the message validator data")
	}
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
	if _, err := p.Encoding().EncodeGossip(buf, exit); err != nil {
		t.Fatal(err)
	}
	m := &pubsub.Message{
		Message: &pubsubpb.Message{
			Data: buf.Bytes(),
			TopicIDs: []string{
				p2p.GossipTypeMapping[reflect.TypeOf(exit)],
			},
		},
	}
	valid := r.validateVoluntaryExit(ctx, "", m) == pubsub.ValidationAccept
	if valid {
		t.Error("Validation should have failed")
	}
}
