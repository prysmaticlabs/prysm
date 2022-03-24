package types

import types "github.com/prysmaticlabs/eth2-types"

// BoostProposerRootArgs to call the BoostProposerRoot function.
type BoostProposerRootArgs struct {
	BlockRoot       [32]byte
	BlockSlot       types.Slot
	CurrentSlot     types.Slot
	SecondsIntoSlot uint64
}
