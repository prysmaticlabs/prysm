package validator

import (
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestProposer_SortProfitableAtts(t *testing.T) {
	atts := proposerAtts([]*ethpb.Attestation{
		{Data: &ethpb.AttestationData{Slot: 4, BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}, AggregationBits: bitfield.Bitlist{0b11100000}},
		{Data: &ethpb.AttestationData{Slot: 1, BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}, AggregationBits: bitfield.Bitlist{0b11000000}},
		{Data: &ethpb.AttestationData{Slot: 2, BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}, AggregationBits: bitfield.Bitlist{0b11100000}},
		{Data: &ethpb.AttestationData{Slot: 4, BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}, AggregationBits: bitfield.Bitlist{0b11110000}},
		{Data: &ethpb.AttestationData{Slot: 1, BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}, AggregationBits: bitfield.Bitlist{0b11100000}},
		{Data: &ethpb.AttestationData{Slot: 3, BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}, AggregationBits: bitfield.Bitlist{0b11000000}},
	})
	want := proposerAtts([]*ethpb.Attestation{
		{Data: &ethpb.AttestationData{Slot: 4, BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}, AggregationBits: bitfield.Bitlist{0b11110000}},
		{Data: &ethpb.AttestationData{Slot: 4, BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}, AggregationBits: bitfield.Bitlist{0b11100000}},
		{Data: &ethpb.AttestationData{Slot: 3, BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}, AggregationBits: bitfield.Bitlist{0b11000000}},
		{Data: &ethpb.AttestationData{Slot: 2, BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}, AggregationBits: bitfield.Bitlist{0b11100000}},
		{Data: &ethpb.AttestationData{Slot: 1, BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}, AggregationBits: bitfield.Bitlist{0b11100000}},
		{Data: &ethpb.AttestationData{Slot: 1, BeaconBlockRoot: make([]byte, 32), Target: &ethpb.Checkpoint{Root: make([]byte, 32)}, Source: &ethpb.Checkpoint{Root: make([]byte, 32)}}, AggregationBits: bitfield.Bitlist{0b11000000}},
	})
	atts = atts.sortByProfitability()
	require.DeepEqual(t, want, atts)
}
