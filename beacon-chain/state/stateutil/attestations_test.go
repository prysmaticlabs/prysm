package stateutil

import (
	"bytes"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestAttestationDataRoot_EqualGeneric(t *testing.T) {
	attData := &ethpb.AttestationData{
		Slot:            39,
		CommitteeIndex:  2,
		BeaconBlockRoot: bytesutil.PadTo([]byte("block root"), 32),
		Source: &ethpb.Checkpoint{
			Root:  bytesutil.PadTo([]byte("source root"), 32),
			Epoch: 0,
		},
		Target: &ethpb.Checkpoint{
			Root:  bytesutil.PadTo([]byte("target root"), 32),
			Epoch: 9,
		},
	}
	genericHtr, err := attData.HashTreeRoot()
	require.NoError(t, err)
	dataHtr, err := AttestationDataRoot(attData)
	require.NoError(t, err)

	if !bytes.Equal(genericHtr[:], dataHtr[:]) {
		t.Fatalf("Expected %#x, received %#x", genericHtr, dataHtr)
	}
}

func BenchmarkAttestationDataRoot(b *testing.B) {
	attData := &ethpb.AttestationData{
		Slot:            39,
		CommitteeIndex:  2,
		BeaconBlockRoot: bytesutil.PadTo([]byte("block root"), 32),
		Source: &ethpb.Checkpoint{
			Root:  bytesutil.PadTo([]byte("source root"), 32),
			Epoch: 0,
		},
		Target: &ethpb.Checkpoint{
			Root:  bytesutil.PadTo([]byte("target root"), 32),
			Epoch: 9,
		},
	}
	b.Run("generic", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := attData.HashTreeRoot()
			require.NoError(b, err)
		}
	})
	b.Run("stateutil", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := AttestationDataRoot(attData)
			require.NoError(b, err)
		}
	})
}
