package sync

import (
	"context"
	"reflect"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
)

func TestBeaconAggregateProofSubscriber_CanSave(t *testing.T) {
	r := &Service{
		attPool: attestations.NewPool(),
	}

	a := &ethpb.AggregateAttestationAndProof{Aggregate: &ethpb.Attestation{AggregationBits: bitfield.Bitlist{0x07}}, AggregatorIndex: 100}
	if err := r.beaconAggregateProofSubscriber(context.Background(), a); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(r.attPool.AggregatedAttestations(), []*ethpb.Attestation{a.Aggregate}) {
		t.Error("Did not save aggregated attestation")
	}
}
