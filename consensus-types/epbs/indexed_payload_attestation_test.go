package epbs

import (
	"crypto/rand"
	"encoding/binary"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestGetAttestingIndices(t *testing.T) {
	attestingIndices := []primitives.ValidatorIndex{primitives.ValidatorIndex(randomUint64(t)), primitives.ValidatorIndex(randomUint64(t)), primitives.ValidatorIndex(randomUint64(t))}
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
		Slot: primitives.Slot(randomUint64(t)),
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
	sig := randomBytes(32, t)
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

func randomUint64(t *testing.T) uint64 {
	var num uint64
	b := randomBytes(8, t)
	num = binary.BigEndian.Uint64(b)
	return num
}

func randomBytes(n int, t *testing.T) []byte {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		t.Fatalf("Failed to generate random bytes: %v", err)
	}
	return b
}
