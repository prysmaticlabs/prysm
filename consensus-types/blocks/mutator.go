package blocks

import (
	"github.com/prysmaticlabs/prysm/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/runtime/version"
)

type blockMutator struct {
	Phase0    func(beaconBlock *eth.SignedBeaconBlock)
	Altair    func(beaconBlock *eth.SignedBeaconBlockAltair)
	Bellatrix func(beaconBlock *eth.SignedBeaconBlockBellatrix)
}

func (m blockMutator) apply(b interfaces.SignedBeaconBlock) error {
	switch b.Version() {
	case version.Phase0:
		bb, err := b.PbPhase0Block()
		if err != nil {
			return err
		}
		m.Phase0(bb)
		return nil
	case version.Altair:
		bb, err := b.PbAltairBlock()
		if err != nil {
			return err
		}
		m.Altair(bb)
		return nil
	case version.Bellatrix:
		bb, err := b.PbBellatrixBlock()
		if err != nil {
			return err
		}
		m.Bellatrix(bb)
		return nil
	default:
		return errIncorrectBlockVersion
	}
}

// SetBlockStateRoot modifies the block's state root.
func SetBlockStateRoot(b interfaces.SignedBeaconBlock, sr [32]byte) error {
	return blockMutator{
		Phase0:    func(bb *eth.SignedBeaconBlock) { bb.Block.StateRoot = sr[:] },
		Altair:    func(bb *eth.SignedBeaconBlockAltair) { bb.Block.StateRoot = sr[:] },
		Bellatrix: func(bb *eth.SignedBeaconBlockBellatrix) { bb.Block.StateRoot = sr[:] },
	}.apply(b)
}

// SetBlockParentRoot modifies the block's parent root.
func SetBlockParentRoot(b interfaces.SignedBeaconBlock, pr [32]byte) error {
	return blockMutator{
		Phase0:    func(bb *eth.SignedBeaconBlock) { bb.Block.ParentRoot = pr[:] },
		Altair:    func(bb *eth.SignedBeaconBlockAltair) { bb.Block.ParentRoot = pr[:] },
		Bellatrix: func(bb *eth.SignedBeaconBlockBellatrix) { bb.Block.ParentRoot = pr[:] },
	}.apply(b)
}

// SetBlockSlot modifies the block's slot.
func SetBlockSlot(b interfaces.SignedBeaconBlock, s types.Slot) error {
	return blockMutator{
		Phase0:    func(bb *eth.SignedBeaconBlock) { bb.Block.Slot = s },
		Altair:    func(bb *eth.SignedBeaconBlockAltair) { bb.Block.Slot = s },
		Bellatrix: func(bb *eth.SignedBeaconBlockBellatrix) { bb.Block.Slot = s },
	}.apply(b)
}

// SetProposerIndex modifies the block's proposer index.
func SetProposerIndex(b interfaces.SignedBeaconBlock, idx types.ValidatorIndex) error {
	return blockMutator{
		Phase0:    func(bb *eth.SignedBeaconBlock) { bb.Block.ProposerIndex = idx },
		Altair:    func(bb *eth.SignedBeaconBlockAltair) { bb.Block.ProposerIndex = idx },
		Bellatrix: func(bb *eth.SignedBeaconBlockBellatrix) { bb.Block.ProposerIndex = idx },
	}.apply(b)
}
