package stategen

import (
	"context"
)

// This advances the split slot point between the cold and hot sections.
// It moves the new finalized states from the hot section to the cold section.
func migrateToCold(ctx context.Context, root [32]byte) error {
	return nil
}
