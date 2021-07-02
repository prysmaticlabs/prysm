package copyutil

import (
	"testing"

	prysmv2 "github.com/prysmaticlabs/prysm/proto/prysm/v2"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestCopySignedBeaconBlockAltair(t *testing.T) {
	b := &prysmv2.SignedBeaconBlock{
		Block: &prysmv2.BeaconBlock{
			Body: &prysmv2.BeaconBlockBody{
				RandaoReveal: []byte{'a'},
			},
		},
	}
	// Before copy
	notCopied := b
	b.Block.Body.RandaoReveal = []byte{'b'}
	require.DeepSSZEqual(t, notCopied.Block.Body.RandaoReveal, b.Block.Body.RandaoReveal)

	// After copy
	b.Block.Body.RandaoReveal = []byte{'a'}
	copied := CopySignedBeaconBlockAltair(b)
	b.Block.Body.RandaoReveal = []byte{'b'}
	require.DeepNotSSZEqual(t, copied.Block.Body.RandaoReveal, b.Block.Body.RandaoReveal)
}

func TestCopyBeaconBlockAltair(t *testing.T) {
	b := &prysmv2.BeaconBlock{
		Body: &prysmv2.BeaconBlockBody{
			RandaoReveal: []byte{'a'},
		},
	}
	// Before copy
	notCopied := b
	b.Body.RandaoReveal = []byte{'b'}
	require.DeepSSZEqual(t, notCopied.Body.RandaoReveal, b.Body.RandaoReveal)

	// After copy
	b.Body.RandaoReveal = []byte{'a'}
	copied := CopyBeaconBlockAltair(b)
	b.Body.RandaoReveal = []byte{'b'}
	require.DeepNotSSZEqual(t, copied.Body.RandaoReveal, b.Body.RandaoReveal)
}

func TestCopyBeaconBlockBodyAltair(t *testing.T) {
	b := &prysmv2.BeaconBlockBody{
		RandaoReveal: []byte{'a'},
	}
	// Before copy
	notCopied := b
	b.RandaoReveal = []byte{'b'}
	require.DeepSSZEqual(t, notCopied.RandaoReveal, b.RandaoReveal)

	// After copy
	b.RandaoReveal = []byte{'a'}
	copied := CopyBeaconBlockBodyAltair(b)
	b.RandaoReveal = []byte{'b'}
	require.DeepNotSSZEqual(t, copied.RandaoReveal, b.RandaoReveal)
}

func TestCopySyncCommitteeMessage(t *testing.T) {
	s := &prysmv2.SyncCommitteeMessage{
		Slot: 1,
	}
	// Before copy
	notCopied := s
	s.Slot = 2
	require.DeepSSZEqual(t, notCopied.Slot, s.Slot)

	// After copy
	s.Slot = 1
	copied := CopySyncCommitteeMessage(s)
	s.Slot = 2
	require.DeepNotSSZEqual(t, copied.Slot, s.Slot)
}

func TestCopySyncAggregate(t *testing.T) {
	s := &prysmv2.SyncAggregate{
		SyncCommitteeSignature: []byte{'a'},
	}
	// Before copy
	notCopied := s
	s.SyncCommitteeSignature = []byte{'b'}
	require.DeepSSZEqual(t, notCopied.SyncCommitteeSignature, s.SyncCommitteeSignature)

	// After copy
	s.SyncCommitteeSignature = []byte{'a'}
	copied := CopySyncAggregate(s)
	s.SyncCommitteeSignature = []byte{'b'}
	require.DeepNotSSZEqual(t, copied.SyncCommitteeSignature, s.SyncCommitteeSignature)
}
