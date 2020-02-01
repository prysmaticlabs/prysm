package sync

import (
	"context"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestProcessPendingAtts_NoBlock(t *testing.T) {

}

func TestProcessPendingAtts_HasBlockUnAggregatedAtt(t *testing.T) {

}

func TestProcessPendingAtts_HasBlockAggregatedAtt(t *testing.T) {

}

func TestValidatePendingAtts_CanDelete(t *testing.T) {
	db := dbtest.SetupDB(t)
	defer dbtest.TeardownDB(t, db)

	s := &Service{
		blkRootToPendingAtts: make(map[[32]byte][]*ethpb.AggregateAttestationAndProof),
	}

	// 100 Attestations per block root.
	r1 := [32]byte{'A'}
	r2 := [32]byte{'B'}
	r3 := [32]byte{'C'}

	for i := 0; i < 100; i++ {
		s.savePendingAtt(&ethpb.AggregateAttestationAndProof{
			Aggregate: &ethpb.Attestation{
				Data: &ethpb.AttestationData{Slot: uint64(i), BeaconBlockRoot: r1[:]}}})
		s.savePendingAtt(&ethpb.AggregateAttestationAndProof{
			Aggregate: &ethpb.Attestation{
				Data: &ethpb.AttestationData{Slot: uint64(i), BeaconBlockRoot: r2[:]}}})
		s.savePendingAtt(&ethpb.AggregateAttestationAndProof{
			Aggregate: &ethpb.Attestation{
				Data: &ethpb.AttestationData{Slot: uint64(i), BeaconBlockRoot: r3[:]}}})
	}

	if len(s.blkRootToPendingAtts[r1]) != 100 {
		t.Error("Did not save pending atts")
	}
	if len(s.blkRootToPendingAtts[r2]) != 100 {
		t.Error("Did not save pending atts")
	}
	if len(s.blkRootToPendingAtts[r3]) != 100 {
		t.Error("Did not save pending atts")
	}

	// Set current slot to 50, it should prune 19 attestations. (50 - 31)
	s.validatePendingAtts(context.Background(), 50)
	if len(s.blkRootToPendingAtts[r1]) != 81 {
		t.Error("Did not delete pending atts")
	}
	if len(s.blkRootToPendingAtts[r2]) != 81 {
		t.Error("Did not delete pending atts")
	}
	if len(s.blkRootToPendingAtts[r3]) != 81 {
		t.Error("Did not delete pending atts")
	}

	// Set current slot to 100 + slot_duration, it should prune all the attestations.
	s.validatePendingAtts(context.Background(), 100+params.BeaconConfig().SlotsPerEpoch)
	if len(s.blkRootToPendingAtts[r1]) != 0 {
		t.Log(len(s.blkRootToPendingAtts[r1]))
		t.Error("Did not delete pending atts")
	}
	if len(s.blkRootToPendingAtts[r2]) != 0 {
		t.Error("Did not delete pending atts")
	}
	if len(s.blkRootToPendingAtts[r3]) != 0 {
		t.Error("Did not delete pending atts")
	}

	// Verify the keys are deleted.
	if len(s.blkRootToPendingAtts) != 0 {
		t.Error("Did not delete block keys")
	}
}
