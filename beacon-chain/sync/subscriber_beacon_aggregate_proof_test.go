package sync

import (
	"context"
	"testing"

	lru "github.com/hashicorp/golang-lru"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestBeaconAggregateProofSubscriber_CanSaveAggregatedAttestation(t *testing.T) {
	c, err := lru.New(10)
	require.NoError(t, err)
	r := &Service{
		attPool:              attestations.NewPool(),
		seenAttestationCache: c,
		attestationNotifier:  (&mock.ChainService{}).OperationNotifier(),
	}

	a := &ethpb.SignedAggregateAttestationAndProof{Message: &ethpb.AggregateAttestationAndProof{Aggregate: &ethpb.Attestation{Data: &ethpb.AttestationData{Target: &ethpb.Checkpoint{}}, AggregationBits: bitfield.Bitlist{0x07}}, AggregatorIndex: 100}}
	require.NoError(t, r.beaconAggregateProofSubscriber(context.Background(), a))
	assert.DeepEqual(t, []*ethpb.Attestation{a.Message.Aggregate}, r.attPool.AggregatedAttestations(), "Did not save aggregated attestation")
}

func TestBeaconAggregateProofSubscriber_CanSaveUnaggregatedAttestation(t *testing.T) {
	c, err := lru.New(10)
	require.NoError(t, err)
	r := &Service{
		attPool:              attestations.NewPool(),
		seenAttestationCache: c,
		attestationNotifier:  (&mock.ChainService{}).OperationNotifier(),
	}

	a := &ethpb.SignedAggregateAttestationAndProof{
		Message: &ethpb.AggregateAttestationAndProof{
			Aggregate: &ethpb.Attestation{
				Data: &ethpb.AttestationData{
					Target: &ethpb.Checkpoint{
						Root: make([]byte, 32),
					},
					Source: &ethpb.Checkpoint{
						Root: make([]byte, 32),
					},
					BeaconBlockRoot: make([]byte, 32),
				},
				AggregationBits: bitfield.Bitlist{0x03},
				Signature: make([]byte, 96),
			},
			AggregatorIndex: 100,
		},
	}
	require.NoError(t, r.beaconAggregateProofSubscriber(context.Background(), a))
	assert.DeepEqual(t, []*ethpb.Attestation{a.Message.Aggregate}, r.attPool.UnaggregatedAttestations(), "Did not save unaggregated attestation")
}
