package sync

import (
	"context"
	"testing"

	lru "github.com/hashicorp/golang-lru"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestBeaconAggregateProofSubscriber_CanSaveAggregatedAttestation(t *testing.T) {
	c, err := lru.New(10)
	require.NoError(t, err)
	r := &Service{
		cfg: &Config{
			AttPool:             attestations.NewPool(),
			AttestationNotifier: (&mock.ChainService{}).OperationNotifier(),
		},
		seenAttestationCache: c,
	}

	a := &ethpb.SignedAggregateAttestationAndProof{
		Message: &ethpb.AggregateAttestationAndProof{
			Aggregate: testutil.HydrateAttestation(&ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0x07},
			}),
			AggregatorIndex: 100,
		},
		Signature: make([]byte, 96),
	}
	require.NoError(t, r.beaconAggregateProofSubscriber(context.Background(), a))
	assert.DeepEqual(t, []*ethpb.Attestation{a.Message.Aggregate}, r.cfg.AttPool.AggregatedAttestations(), "Did not save aggregated attestation")
}

func TestBeaconAggregateProofSubscriber_CanSaveUnaggregatedAttestation(t *testing.T) {
	c, err := lru.New(10)
	require.NoError(t, err)
	r := &Service{
		cfg: &Config{
			AttPool:             attestations.NewPool(),
			AttestationNotifier: (&mock.ChainService{}).OperationNotifier(),
		},
		seenAttestationCache: c,
	}

	a := &ethpb.SignedAggregateAttestationAndProof{
		Message: &ethpb.AggregateAttestationAndProof{
			Aggregate: testutil.HydrateAttestation(&ethpb.Attestation{
				AggregationBits: bitfield.Bitlist{0x03},
				Signature:       make([]byte, 96),
			}),
			AggregatorIndex: 100,
		},
	}
	require.NoError(t, r.beaconAggregateProofSubscriber(context.Background(), a))

	atts, err := r.cfg.AttPool.UnaggregatedAttestations()
	require.NoError(t, err)
	assert.DeepEqual(t, []*ethpb.Attestation{a.Message.Aggregate}, atts, "Did not save unaggregated attestation")
}
