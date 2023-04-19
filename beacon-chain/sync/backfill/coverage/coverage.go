package coverage

import "github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"

type AvailableBlocker interface {
	AvailableBlock(primitives.Slot) bool
}
