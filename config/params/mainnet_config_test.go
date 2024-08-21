package params

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestMaxRequestBlock(t *testing.T) {
	testCases := []struct {
		epoch            primitives.Epoch
		expectedMaxBlock uint64
		description      string
	}{
		{
			epoch:            primitives.Epoch(mainnetDenebForkEpoch - 1), // Assuming the fork epoch is not 0
			expectedMaxBlock: mainnetBeaconConfig.MaxRequestBlocks,
		},
		{
			epoch:            primitives.Epoch(mainnetDenebForkEpoch),
			expectedMaxBlock: mainnetBeaconConfig.MaxRequestBlocksDeneb,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			maxBlocks := MaxRequestBlock(tc.epoch)
			if maxBlocks != tc.expectedMaxBlock {
				t.Errorf("For epoch %d, expected max blocks %d, got %d", tc.epoch, tc.expectedMaxBlock, maxBlocks)
			}
		})
	}
}

func TestComputeGenesisValidatorsRoot(t *testing.T) {
	var wantRoot [32]byte = [32]byte{
		75, 54, 61, 185, 78, 40, 97, 32, 215, 110, 185, 5, 52, 15, 221, 78,
		84, 191, 233, 240, 107, 243, 63, 246, 207, 90, 210, 127, 81, 27, 254, 149,
	}
	gotRoot := ComputeGenesisValidatorsRoot("0x4b363db94e286120d76eb905340fdd4e54bfe9f06bf33ff6cf5ad27f511bfe95")
	require.DeepEqual(t, gotRoot, wantRoot)
}
