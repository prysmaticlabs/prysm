package eth_test

import (
	"fmt"
	"testing"

	fuzz "github.com/google/gofuzz"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestCopyExecutionPayloadHeader_Fuzz(t *testing.T) {
	fuzzCopies(t, &eth.Checkpoint{})
	fuzzCopies(t, &eth.AttestationData{})
	fuzzCopies(t, &eth.Attestation{})
	fuzzCopies(t, &eth.AttestationElectra{})
	fuzzCopies(t, &eth.PendingAttestation{})
	fuzzCopies(t, &eth.IndexedAttestation{})
	fuzzCopies(t, &eth.IndexedAttestationElectra{})
	fuzzCopies(t, &eth.AttesterSlashing{})
	fuzzCopies(t, &eth.AttesterSlashingElectra{})
}

func fuzzCopies[T any, C eth.Copier[T]](t *testing.T, obj C) {
	fuzzer := fuzz.NewWithSeed(0)
	amount := 1000
	t.Run(fmt.Sprintf("%T", obj), func(t *testing.T) {
		for i := 0; i < amount; i++ {
			fuzzer.Fuzz(obj) // Populate thing with random values
			got := obj.Copy()
			require.DeepEqual(t, obj, got)
			// check shallow copy working
			fuzzer.Fuzz(got)
			require.DeepNotEqual(t, obj, got)
			// TODO: think of deeper not equal fuzzing
		}
	})
}
