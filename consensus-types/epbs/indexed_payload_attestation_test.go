package epbs

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestGetAttestingIndices(t *testing.T) {
	attestingIndices := []primitives.ValidatorIndex{1, 2, 3}
	nilAttestingIndices := []primitives.ValidatorIndex{}
	pa := &IndexedPayloadAttestation{
		AttestingIndices: attestingIndices,
	}
	got := pa.GetAttestingIndices()
	for i, v := range got {
		require.Equal(t, attestingIndices[i], v)
	}
	pa = &IndexedPayloadAttestation{
		AttestingIndices: nilAttestingIndices,
	}
	got = pa.GetAttestingIndices()
	require.DeepEqual(t, nilAttestingIndices, got)
}

func TestGetData(t *testing.T) {
	data := &eth.PayloadAttestationData{
		Slot: 1,
	}
	nilData := &eth.PayloadAttestationData{}
	pa := &IndexedPayloadAttestation{
		Data: data,
	}
	got := pa.GetData()
	require.Equal(t, data, got)

	pa = &IndexedPayloadAttestation{
		Data: nilData,
	}
	got = pa.GetData()
	require.Equal(t, nilData, got)
}

func TestGetSignature(t *testing.T) {
	sig := []byte{2}
	nilSig := []byte{}
	pa := &IndexedPayloadAttestation{
		Signature: sig,
	}
	got := pa.GetSignature()
	require.DeepEqual(t, sig, got)

	pa = &IndexedPayloadAttestation{
		Signature: nilSig,
	}
	got = pa.GetSignature()
	require.DeepEqual(t, nilSig, got)
}
