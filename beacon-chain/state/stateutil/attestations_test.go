package stateutil

import (
	"bytes"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
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
	genericHtr, err := ssz.HashTreeRoot(attData)
	if err != nil {
		t.Fatal(err)
	}
	dataHtr, err := AttestationDataRoot(attData)
	if err != nil {
		t.Fatal(err)
	}
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
			_, err := ssz.HashTreeRoot(attData)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("stateutil", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := AttestationDataRoot(attData)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
