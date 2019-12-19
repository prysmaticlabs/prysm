package sync

import (
	"bytes"
	"context"
	"crypto/rand"
	"reflect"
	"testing"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsubpb "github.com/libp2p/go-libp2p-pubsub/pb"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func setupValidExit(t *testing.T) (*ethpb.VoluntaryExit, *pb.BeaconState) {
	exit := &ethpb.VoluntaryExit{
		ValidatorIndex: 0,
		Epoch:          0,
	}
	registry := []*ethpb.Validator{
		{
			ExitEpoch:       params.BeaconConfig().FarFutureEpoch,
			ActivationEpoch: 0,
		},
	}
	state := &pb.BeaconState{
		Validators: registry,
		Fork: &pb.Fork{
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
		},
		Slot: params.BeaconConfig().SlotsPerEpoch * 5,
	}
	state.Slot = state.Slot + (params.BeaconConfig().PersistentCommitteePeriod * params.BeaconConfig().SlotsPerEpoch)
	signingRoot, err := ssz.SigningRoot(exit)
	if err != nil {
		t.Error(err)
	}
	domain := helpers.Domain(state.Fork, helpers.CurrentEpoch(state), params.BeaconConfig().DomainVoluntaryExit)
	priv := bls.RandKey()

	sig := priv.Sign(signingRoot[:], domain)
	exit.Signature = sig.Marshal()
	state.Validators[0].PublicKey = priv.PublicKey().Marshal()[:]

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

	r := &Service{
		p2p: p,
		chain: &mock.ChainService{
			State: s,
		},
		initialSync: &mockSync.Sync{IsSyncing: false},
	}

	buf := new(bytes.Buffer)
	if _, err := p.Encoding().Encode(buf, exit); err != nil {
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
	valid := r.validateVoluntaryExit(ctx, "", m)
	if !valid {
		t.Error("Failed validation")
	}
}

func TestValidateVoluntaryExit_ValidExit_FromSelf(t *testing.T) {
	p := p2ptest.NewTestP2P(t)
	ctx := context.Background()

	exit, s := setupValidExit(t)

	r := &Service{
		p2p: p,
		chain: &mock.ChainService{
			State: s,
		},
		initialSync: &mockSync.Sync{IsSyncing: false},
	}
	buf := new(bytes.Buffer)
	if _, err := p.Encoding().Encode(buf, exit); err != nil {
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
	valid := r.validateVoluntaryExit(ctx, p.PeerID(), m)
	if valid {
		t.Error("Validation should have failed")
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
	if _, err := p.Encoding().Encode(buf, exit); err != nil {
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
	valid := r.validateVoluntaryExit(ctx, "", m)
	if valid {
		t.Error("Validation should have failed")
	}
}
