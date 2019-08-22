package sync

import (
	"context"
	"math/rand"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func setupValidAttesterSlashing(t *testing.T, db db.Database) *ethpb.AttesterSlashing {
	ctx := context.Background()
	deposits, privKeys := testutil.SetupInitialDeposits(t, 5)
	beaconState, err := state.GenesisBeaconState(deposits, 0, &ethpb.Eth1Data{})
	for _, vv := range beaconState.Validators {
		vv.WithdrawableEpoch = 1 * params.BeaconConfig().SlotsPerEpoch
	}

	att1 := &ethpb.IndexedAttestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 1},
			Target: &ethpb.Checkpoint{Epoch: 0},
			Crosslink: &ethpb.Crosslink{
				Shard: 4,
			},
		},
		CustodyBit_0Indices: []uint64{0, 1},
	}
	dataAndCustodyBit := &pb.AttestationDataAndCustodyBit{
		Data:       att1.Data,
		CustodyBit: false,
	}
	hashTreeRoot, err := ssz.HashTreeRoot(dataAndCustodyBit)
	if err != nil {
		t.Error(err)
	}
	domain := helpers.Domain(beaconState, 0, params.BeaconConfig().DomainAttestation)
	sig0 := privKeys[0].Sign(hashTreeRoot[:], domain)
	sig1 := privKeys[1].Sign(hashTreeRoot[:], domain)
	aggregateSig := bls.AggregateSignatures([]*bls.Signature{sig0, sig1})
	att1.Signature = aggregateSig.Marshal()[:]

	att2 := &ethpb.IndexedAttestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0},
			Target: &ethpb.Checkpoint{Epoch: 0},
			Crosslink: &ethpb.Crosslink{
				Shard: 4,
			},
		},
		CustodyBit_0Indices: []uint64{0, 1},
	}
	dataAndCustodyBit = &pb.AttestationDataAndCustodyBit{
		Data:       att2.Data,
		CustodyBit: false,
	}
	hashTreeRoot, err = ssz.HashTreeRoot(dataAndCustodyBit)
	if err != nil {
		t.Error(err)
	}
	sig0 = privKeys[0].Sign(hashTreeRoot[:], domain)
	sig1 = privKeys[1].Sign(hashTreeRoot[:], domain)
	aggregateSig = bls.AggregateSignatures([]*bls.Signature{sig0, sig1})
	att2.Signature = aggregateSig.Marshal()[:]

	slashing := &ethpb.AttesterSlashing{
		Attestation_1: att1,
		Attestation_2: att2,
	}

	currentSlot := 2 * params.BeaconConfig().SlotsPerEpoch
	beaconState.Slot = currentSlot

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	headBlockRoot := bytesutil.ToBytes32(b)
	if err := db.SaveState(ctx, beaconState, headBlockRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveHeadBlockRoot(ctx, headBlockRoot); err != nil {
		t.Fatal(err)
	}
	return slashing
}

func TestValidateAttesterSlashing_ValidSlashing(t *testing.T) {
	db := dbtest.SetupDB(t)
	defer dbtest.TeardownDB(t, db)
	p2p := p2ptest.NewTestP2P(t)
	ctx := context.Background()

	slashing := setupValidAttesterSlashing(t, db)

	r := &RegularSync{
		p2p: p2p,
		db:  db,
	}

	if !r.validateAttesterSlashing(ctx, slashing, p2p) {
		t.Error("Failed validation")
	}

	if !p2p.BroadcastCalled {
		t.Error("Broadcast was not called")
	}

	// A second message with the same information should not be valid for processing or
	// propagation.
	p2p.BroadcastCalled = false
	if r.validateAttesterSlashing(ctx, slashing, p2p) {
		t.Error("Passed validation when should have failed")
	}

	if p2p.BroadcastCalled {
		t.Error("broadcast was called when it should not have been called")
	}
}
