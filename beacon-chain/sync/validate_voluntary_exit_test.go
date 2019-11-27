package sync

import (
	"context"
	"crypto/rand"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
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
	priv, err := bls.RandKey(rand.Reader)
	if err != nil {
		t.Error(err)
	}
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
	p2p := p2ptest.NewTestP2P(t)
	ctx := context.Background()

	exit, s := setupValidExit(t)

	r := &RegularSync{
		p2p: p2p,
		chain: &mock.ChainService{
			State: s,
		},
		initialSync: &mockSync.Sync{IsSyncing: false},
	}

	valid, err := r.validateVoluntaryExit(ctx, exit, p2p, false /*fromSelf*/)
	if err != nil {
		t.Errorf("Failed validation: %v", err)
	}
	if !valid {
		t.Error("Failed validation")
	}

	if !p2p.BroadcastCalled {
		t.Error("Broadcast was not called")
	}

	// A second message with the same information should not be valid for processing or
	// propagation.
	p2p.BroadcastCalled = false
	valid, _ = r.validateVoluntaryExit(ctx, exit, p2p, false /*fromSelf*/)
	if valid {
		t.Error("Passed validation when should have failed")
	}

	if p2p.BroadcastCalled {
		t.Error("broadcast was called when it should not have been called")
	}
}

func TestValidateVoluntaryExit_ValidExit_FromSelf(t *testing.T) {
	p2p := p2ptest.NewTestP2P(t)
	ctx := context.Background()

	exit, s := setupValidExit(t)

	r := &RegularSync{
		p2p: p2p,
		chain: &mock.ChainService{
			State: s,
		},
		initialSync: &mockSync.Sync{IsSyncing: false},
	}

	valid, _ := r.validateVoluntaryExit(ctx, exit, p2p, true /*fromSelf*/)
	if valid {
		t.Error("Validation should have failed")
	}

	if p2p.BroadcastCalled {
		t.Error("Broadcast was called")
	}
}

func TestValidateVoluntaryExit_ValidExit_Syncing(t *testing.T) {
	p2p := p2ptest.NewTestP2P(t)
	ctx := context.Background()

	exit, s := setupValidExit(t)

	r := &RegularSync{
		p2p: p2p,
		chain: &mock.ChainService{
			State: s,
		},
		initialSync: &mockSync.Sync{IsSyncing: true},
	}

	valid, _ := r.validateVoluntaryExit(ctx, exit, p2p, false /*fromSelf*/)
	if valid {
		t.Error("Validation should have failed")
	}

	if p2p.BroadcastCalled {
		t.Error("Broadcast was called")
	}
}
