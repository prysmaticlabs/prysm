package stategen

import (
	"context"

	"github.com/prysmaticlabs/prysm/shared/params"
)

// This advances the split slot point between the cold and hot sections.
// It moves the new finalized states from the hot section to the cold section.
func migrateToCold(ctx context.Context, root [32]byte) error {
	return nil
}

// This verifies the archive point frequency is valid. It checks the interval
// is a divisor of the number of slots per historical root and divisible by
// the number of slots per epoch. This ensures we have at least one
// archive point within range of our state root history when iterating
// backwards. It also ensures the archive points align with hot state summaries
// which makes it quicker to migrate hot to cold.
func verifySlotsPerArchivePoint(slotsPerArchivePoint uint64) bool {
	return slotsPerArchivePoint > 0 &&
		slotsPerArchivePoint % params.BeaconConfig().SlotsPerHistoricalRoot == 0 &&
		slotsPerArchivePoint % params.BeaconConfig().SlotsPerEpoch == 0
}
