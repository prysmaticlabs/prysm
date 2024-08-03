package epbs

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestGetAttestingIndices(t *testing.T) {
	attestingIndices := []primitives.ValidatorIndex{1, 2, 3}
	nilAttestingIndices := []primitives.ValidatorIndex(nil)
	pa := &IndexedPayloadAttestation{
		AttestingIndices: attestingIndices,
	}
	got := pa.GetAttestingIndices()
	require.DeepEqual(t, attestingIndices, got)
	pa = nil
	got = pa.GetAttestingIndices()
	require.DeepEqual(t, nilAttestingIndices, got)
}

func TestGetData(t *testing.T) {
	data := &eth.PayloadAttestationData{
		Slot: 1,
	}
	nilData := (*eth.PayloadAttestationData)(nil)
	pa := &IndexedPayloadAttestation{
		Data: data,
	}
	got := pa.GetData()
	require.Equal(t, data, got)

	pa = nil
	got = pa.GetData()
	require.DeepEqual(t, got, nilData)
}

func TestGetSignature(t *testing.T) {
	sig := []byte{2}
	nilSig := []byte(nil)
	pa := &IndexedPayloadAttestation{
		Signature: sig,
	}
	got := pa.GetSignature()
	require.DeepEqual(t, sig, got)

	pa = nil
	got = pa.GetSignature()
	require.DeepEqual(t, got, nilSig)
}
