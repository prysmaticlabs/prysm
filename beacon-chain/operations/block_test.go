package operations

import (
	"context"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	dbutil "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestReceiveBlkRemoveOps_Ok(t *testing.T) {
	beaconDB := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, beaconDB)
	s := NewService(context.Background(), &Config{BeaconDB: beaconDB})

	attestations := make([]*ethpb.Attestation, 10)
	for i := 0; i < len(attestations); i++ {
		attestations[i] = &ethpb.Attestation{
			Data: &ethpb.AttestationData{Slot: uint64(i),
				Source: &ethpb.Checkpoint{},
				Target: &ethpb.Checkpoint{}},
			AggregationBits: bitfield.Bitlist{0b11},
		}
		if err := s.beaconDB.SaveAttestation(context.Background(), attestations[i]); err != nil {
			t.Fatalf("Failed to save attestation: %v", err)
		}
	}

	headBlockRoot := [32]byte{1, 2, 3}
	if err := beaconDB.SaveHeadBlockRoot(context.Background(), headBlockRoot); err != nil {
		t.Fatal(err)
	}
	if err := beaconDB.SaveState(context.Background(), &pb.BeaconState{Slot: 15}, headBlockRoot); err != nil {
		t.Fatal(err)
	}

	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			Attestations: attestations,
		},
	}

	s.incomingProcessedBlock <- block
	if err := s.handleProcessedBlock(context.Background(), block); err != nil {
		t.Error(err)
	}

	atts, err := s.AttestationPool(context.Background(), 15)
	if err != nil {
		t.Fatal(err)
	}
	if len(atts) != 0 {
		t.Errorf("Attestation pool should be empty but got a length of %d", len(atts))
	}
}
