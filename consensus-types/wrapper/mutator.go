package wrapper

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/runtime/version"
)

type BlockMutator struct {
	Phase0    func(beaconBlock *eth.SignedBeaconBlock)
	Altair    func(beaconBlock *eth.SignedBeaconBlockAltair)
	Bellatrix func(beaconBlock *eth.SignedBeaconBlockBellatrix)
}

func (m BlockMutator) Apply(b interfaces.SignedBeaconBlock) error {
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
	}
	msg := fmt.Sprintf("version %d = %s", b.Version(), version.String(b.Version()))
	return errors.Wrap(ErrUnsupportedSignedBeaconBlock, msg)
}

func SetBlockStateRoot(b interfaces.SignedBeaconBlock, sr [32]byte) error {
	return BlockMutator{
		Phase0:    func(bb *eth.SignedBeaconBlock) { bb.Block.StateRoot = sr[:] },
		Altair:    func(bb *eth.SignedBeaconBlockAltair) { bb.Block.StateRoot = sr[:] },
		Bellatrix: func(bb *eth.SignedBeaconBlockBellatrix) { bb.Block.StateRoot = sr[:] },
	}.Apply(b)
}

func SetBlockParentRoot(b interfaces.SignedBeaconBlock, pr [32]byte) error {
	return BlockMutator{
		Phase0:    func(bb *eth.SignedBeaconBlock) { bb.Block.ParentRoot = pr[:] },
		Altair:    func(bb *eth.SignedBeaconBlockAltair) { bb.Block.ParentRoot = pr[:] },
		Bellatrix: func(bb *eth.SignedBeaconBlockBellatrix) { bb.Block.ParentRoot = pr[:] },
	}.Apply(b)
}

func SetBlockSlot(b interfaces.SignedBeaconBlock, s types.Slot) error {
	return BlockMutator{
		Phase0:    func(bb *eth.SignedBeaconBlock) { bb.Block.Slot = s },
		Altair:    func(bb *eth.SignedBeaconBlockAltair) { bb.Block.Slot = s },
		Bellatrix: func(bb *eth.SignedBeaconBlockBellatrix) { bb.Block.Slot = s },
	}.Apply(b)
}

func SetProposerIndex(b interfaces.SignedBeaconBlock, idx types.ValidatorIndex) error {
	return BlockMutator{
		Phase0:    func(bb *eth.SignedBeaconBlock) { bb.Block.ProposerIndex = idx },
		Altair:    func(bb *eth.SignedBeaconBlockAltair) { bb.Block.ProposerIndex = idx },
		Bellatrix: func(bb *eth.SignedBeaconBlockBellatrix) { bb.Block.ProposerIndex = idx },
	}.Apply(b)
}
