package sync

import (
	"context"
	"crypto/rand"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestValidateVoluntaryExit_ValidExit(t *testing.T) {
	db := dbtest.SetupDB(t)
	defer dbtest.TeardownDB(t, db)
	p2p := p2ptest.NewTestP2P(t)
	ctx := context.Background()

	// Setup a valid exit and state
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
	domain := helpers.Domain(state, helpers.CurrentEpoch(state), params.BeaconConfig().DomainVoluntaryExit)
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
	headBlockRoot := bytesutil.ToBytes32(b)
	if err := db.SaveState(ctx, state, headBlockRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveHeadBlockRoot(ctx, headBlockRoot); err != nil {
		t.Fatal(err)
	}

	r := &RegularSync{
		p2p: p2p,
		db:  db,
	}

	if !r.validateVoluntaryExit(ctx, exit, p2p) {
		t.Error("failed validation")
	}
}
