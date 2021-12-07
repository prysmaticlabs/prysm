package sniff

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/state/v1"
	v2 "github.com/prysmaticlabs/prysm/beacon-chain/state/v2"
	v3 "github.com/prysmaticlabs/prysm/beacon-chain/state/v3"
	"github.com/prysmaticlabs/prysm/config/params"
	v1alpha1 "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/time/slots"
)

// is an io.Reader. This allows client code to remain agnostic about whether the data comes
// from the network or a file without needing to read the entire state into mem as a large byte slice.
func BeaconStateForConfigFork(b []byte, cf *ConfigFork) (state.BeaconState, error) {
	switch cf.Fork {
	case params.ForkGenesis:
		return v1.InitializeFromSSZBytes(b)
	case params.ForkAltair:
		return v2.InitializeFromSSZBytes(b)
	case params.ForkMerge:
		return v3.InitializeFromSSZBytes(b)
	}
	return nil, fmt.Errorf("unable to initialize BeaconState for fork version=%s", cf.Fork.String())
}

// BlockForConfigFork attempts to unmarshal a block from a marshaled byte slice into the correct block type.
// In order to do this it needs to know what fork the block is from using ConfigFork, which can be obtained
// by using ConfigForkForState.
func BlockForConfigFork(b []byte, cf *ConfigFork) (block.SignedBeaconBlock, error) {
	slot, err := SlotFromBlock(b)
	if err != nil {
		return nil, err
	}

	// heuristic to make sure block is from the same version as the state
	// based on the fork schedule for the Config detected from the state
	// get the version that corresponds to the epoch the block is from according to the fork choice schedule
	// and make sure that the version is the same one that was pulled from the state
	epoch := slots.ToEpoch(slot)
	fs := cf.Config.OrderedForkSchedule()
	ver, err := fs.VersionForEpoch(epoch)
	if err != nil {
		return nil, err
	}
	if ver != cf.Version {
		return nil, fmt.Errorf("cannot sniff block schema, block (slot=%d, epoch=%d) is on a different fork", slot, epoch)
	}

	switch cf.Fork {
	case params.ForkGenesis:
		blk := &v1alpha1.SignedBeaconBlock{}
		err := blk.UnmarshalSSZ(b)
		if err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal SignedBeaconBlock in BlockFromSSZReader")
		}
		return wrapper.WrappedPhase0SignedBeaconBlock(blk), nil
	case params.ForkAltair:
		blk := &v1alpha1.SignedBeaconBlockAltair{}
		err := blk.UnmarshalSSZ(b)
		if err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal SignedBeaconBlockAltair in BlockFromSSZReader")
		}
		return wrapper.WrappedAltairSignedBeaconBlock(blk)
	case params.ForkMerge:
		blk := &v1alpha1.SignedBeaconBlockMerge{}
		err := blk.UnmarshalSSZ(b)
		if err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal SignedBeaconBlockMerge in BlockFromSSZReader")
		}
		return wrapper.WrappedMergeSignedBeaconBlock(blk)
	}
	return nil, fmt.Errorf("unable to initialize BeaconBlock for fork version=%s at slot=%d", cf.Fork.String(), slot)
}
