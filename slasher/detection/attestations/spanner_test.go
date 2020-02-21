package attestations

import (
	"testing"
)

func TestSpanDetector_UpdateMinSpan(t *testing.T) {
	type spanMapTestStruct struct {
		validatorIdx   uint64
		sourceEpoch    uint64
		targetEpoch    uint64
		slashableEpoch uint64
		resultSpanMap  [2]uint16
	}
	tests := []spanMapTestStruct{
		{
			validatorIdx:   0,
			sourceEpoch:    3,
			targetEpoch:    6,
			slashableEpoch: 0,
			//resultSpanMap: &slashpb.EpochSpanMap{
			//	EpochSpanMap: map[uint64]*slashpb.MinMaxEpochSpan{
			//		4: {MinEpochSpan: 0, MaxEpochSpan: 2},
			//		5: {MinEpochSpan: 0, MaxEpochSpan: 1},
			//	},
			//},
		},
	}
	_ = tests
}
