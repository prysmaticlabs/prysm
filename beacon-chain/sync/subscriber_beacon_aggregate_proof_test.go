package sync

import (
	"context"
	"reflect"
	"testing"

	lru "github.com/hashicorp/golang-lru"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
)

func TestBeaconAggregateProofSubscriber_CanSave(t *testing.T) {
	c, err := lru.New(10)
	if err != nil {
		t.Fatal(err)
	}
	r := &Service{
		attPool:              attestations.NewPool(),
		seenAttestationCache: c,
	}

	a := &ethpb.SignedAggregateAttestationAndProof{Message: &ethpb.AggregateAttestationAndProof{Aggregate: &ethpb.Attestation{Data: &ethpb.AttestationData{Target: &ethpb.Checkpoint{}}, AggregationBits: bitfield.Bitlist{0x07}}, AggregatorIndex: 100}}
	if err := r.beaconAggregateProofSubscriber(context.Background(), a); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(r.attPool.AggregatedAttestations(), []*ethpb.Attestation{a.Message.Aggregate}) {
		t.Error("Did not save aggregated attestation")
	}
}
