package sync

import (
	"reflect"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
)

func TestDeleteAttsInPool(t *testing.T) {
	r := &Service{
		attPool: attestations.NewPool(),
	}
	data := &ethpb.AttestationData{}
	att1 := &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b1101}, Data: data}
	att2 := &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b1110}, Data: data}
	att3 := &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b1011}, Data: data}
	att4 := &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0b1001}, Data: data}
	if err := r.attPool.SaveAggregatedAttestation(att1); err != nil {
		t.Fatal(err)
	}
	if err := r.attPool.SaveAggregatedAttestation(att2); err != nil {
		t.Fatal(err)
	}
	if err := r.attPool.SaveAggregatedAttestation(att3); err != nil {
		t.Fatal(err)
	}
	if err := r.attPool.SaveUnaggregatedAttestation(att4); err != nil {
		t.Fatal(err)
	}

	// Seen 1, 3 and 4 in block.
	if err := r.deleteAttsInPool([]*ethpb.Attestation{att1, att3, att4}); err != nil {
		t.Fatal(err)
	}

	// Only 2 should remain.
	if !reflect.DeepEqual(r.attPool.AggregatedAttestations(), []*ethpb.Attestation{att2}) {
		t.Error("Did not get wanted attestation from pool")
	}
}
