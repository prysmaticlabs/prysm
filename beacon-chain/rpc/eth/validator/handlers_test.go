package validator

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/operations/attestations"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpbv1 "github.com/prysmaticlabs/prysm/v4/proto/eth/v1"
	ethpbalpha "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func TestGetAggregateAttestation(t *testing.T) {
	ctx := context.Background()
	root1 := bytesutil.PadTo([]byte("root1"), 32)
	sig1 := bytesutil.PadTo([]byte("sig1"), fieldparams.BLSSignatureLength)
	attSlot1 := &ethpbalpha.Attestation{
		AggregationBits: []byte{0, 1},
		Data: &ethpbalpha.AttestationData{
			Slot:            1,
			CommitteeIndex:  1,
			BeaconBlockRoot: root1,
			Source: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root1,
			},
			Target: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root1,
			},
		},
		Signature: sig1,
	}
	root21 := bytesutil.PadTo([]byte("root2_1"), 32)
	sig21 := bytesutil.PadTo([]byte("sig2_1"), fieldparams.BLSSignatureLength)
	attslot21 := &ethpbalpha.Attestation{
		AggregationBits: []byte{0, 1, 1},
		Data: &ethpbalpha.AttestationData{
			Slot:            2,
			CommitteeIndex:  2,
			BeaconBlockRoot: root21,
			Source: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root21,
			},
			Target: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root21,
			},
		},
		Signature: sig21,
	}
	root22 := bytesutil.PadTo([]byte("root2_2"), 32)
	sig22 := bytesutil.PadTo([]byte("sig2_2"), fieldparams.BLSSignatureLength)
	attslot22 := &ethpbalpha.Attestation{
		AggregationBits: []byte{0, 1, 1, 1},
		Data: &ethpbalpha.AttestationData{
			Slot:            2,
			CommitteeIndex:  3,
			BeaconBlockRoot: root22,
			Source: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root22,
			},
			Target: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root22,
			},
		},
		Signature: sig22,
	}
	root33 := bytesutil.PadTo([]byte("root3_3"), 32)
	sig33 := bls.NewAggregateSignature().Marshal()

	attslot33 := &ethpbalpha.Attestation{
		AggregationBits: []byte{1, 0, 0, 1},
		Data: &ethpbalpha.AttestationData{
			Slot:            2,
			CommitteeIndex:  3,
			BeaconBlockRoot: root33,
			Source: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root33,
			},
			Target: &ethpbalpha.Checkpoint{
				Epoch: 1,
				Root:  root33,
			},
		},
		Signature: sig33,
	}
	pool := attestations.NewPool()
	err := pool.SaveAggregatedAttestations([]*ethpbalpha.Attestation{attSlot1, attslot21, attslot22})
	assert.NoError(t, err)
	vs := &Server{
		AttestationsPool: pool,
	}

	t.Run("OK", func(t *testing.T) {
		reqRoot, err := attslot22.Data.HashTreeRoot()
		require.NoError(t, err)
		req := &ethpbv1.AggregateAttestationRequest{
			AttestationDataRoot: reqRoot[:],
			Slot:                2,
		}
		att, err := vs.GetAggregateAttestation(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, att)
		require.NotNil(t, att.Data)
		assert.DeepEqual(t, bitfield.Bitlist{0, 1, 1, 1}, att.Data.AggregationBits)
		assert.DeepEqual(t, sig22, att.Data.Signature)
		assert.Equal(t, primitives.Slot(2), att.Data.Data.Slot)
		assert.Equal(t, primitives.CommitteeIndex(3), att.Data.Data.Index)
		assert.DeepEqual(t, root22, att.Data.Data.BeaconBlockRoot)
		require.NotNil(t, att.Data.Data.Source)
		assert.Equal(t, primitives.Epoch(1), att.Data.Data.Source.Epoch)
		assert.DeepEqual(t, root22, att.Data.Data.Source.Root)
		require.NotNil(t, att.Data.Data.Target)
		assert.Equal(t, primitives.Epoch(1), att.Data.Data.Target.Epoch)
		assert.DeepEqual(t, root22, att.Data.Data.Target.Root)
	})

	t.Run("Aggregate Beforehand", func(t *testing.T) {
		reqRoot, err := attslot33.Data.HashTreeRoot()
		require.NoError(t, err)
		err = vs.AttestationsPool.SaveUnaggregatedAttestation(attslot33)
		require.NoError(t, err)
		newAtt := ethpbalpha.CopyAttestation(attslot33)
		newAtt.AggregationBits = []byte{0, 1, 0, 1}
		err = vs.AttestationsPool.SaveUnaggregatedAttestation(newAtt)
		require.NoError(t, err)
		req := &ethpbv1.AggregateAttestationRequest{
			AttestationDataRoot: reqRoot[:],
			Slot:                2,
		}
		aggAtt, err := vs.GetAggregateAttestation(ctx, req)
		require.NoError(t, err)
		assert.DeepEqual(t, bitfield.Bitlist{1, 1, 0, 1}, aggAtt.Data.AggregationBits)
	})
	t.Run("No matching attestation", func(t *testing.T) {
		req := &ethpbv1.AggregateAttestationRequest{
			AttestationDataRoot: bytesutil.PadTo([]byte("foo"), 32),
			Slot:                2,
		}
		_, err := vs.GetAggregateAttestation(ctx, req)
		assert.ErrorContains(t, "No matching attestation found", err)
	})
}
