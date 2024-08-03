package epbs

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestGetAttestingIndices(t *testing.T) {
	AttestingIndices := []primitives.ValidatorIndex{1, 2, 3}
	NilAttestingIndices := []primitives.ValidatorIndex(nil)
	pa := &IndexedPayloadAttestation{
		AttestingIndices: AttestingIndices,
	}
	got := pa.GetAttestingIndices()
	for i, v := range got {
		require.Equal(t, AttestingIndices[i], v)
	}
	pa = &IndexedPayloadAttestation{
		AttestingIndices: NilAttestingIndices,
	}
	got = pa.GetAttestingIndices()
	for i, v := range got {
		require.Equal(t, AttestingIndices[i], v)
	}
}

func TestGetData(t *testing.T) {
	Data := &eth.PayloadAttestationData{
		Slot: 1,
	}
	NilData := &eth.PayloadAttestationData{}
	pa := &IndexedPayloadAttestation{
		Data: Data,
	}
	got := pa.GetData()
	require.Equal(t, Data, got)

	pa = &IndexedPayloadAttestation{
		Data: NilData,
	}
	got = pa.GetData()
	require.Equal(t, NilData, got)
}

func TestGetSignature(t *testing.T) {
	sig := []byte{2}
	Nilsig := []byte{}
	pa := &IndexedPayloadAttestation{
		Signature: sig,
	}
	got := pa.GetSignature()
	require.DeepEqual(t, sig, got)

	pa = &IndexedPayloadAttestation{
		Signature: Nilsig,
	}
	got = pa.GetSignature()
	require.DeepEqual(t, Nilsig, got)
}
