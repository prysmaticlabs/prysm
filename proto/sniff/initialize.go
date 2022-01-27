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

// BeaconStateForConfigFork uses metadata from the provided *ConfigFork to pick the right BeaconState schema
// to Unmarshal the value contained in the marshaled argument.
func BeaconStateForConfigFork(marshaled []byte, cf *ConfigFork) (state.BeaconState, error) {
	cv, err := CurrentVersionFromState(marshaled)
	if err != nil {
		return nil, errors.Wrap(err, "in BeaconStateForConfigFork error from CurrentVersionFromState")
	}
	if cv != cf.Version {
		return nil, fmt.Errorf("state fork version mismatch, detected=%#x, expected=%#x", cv, cf.Version)
	}
	var s state.BeaconState
	switch cf.Fork {
	case params.ForkGenesis:
		s, err = v1.InitializeFromSSZBytes(marshaled)
		if err != nil {
			return nil, errors.Wrap(err, "InitializeFromSSZBytes for ForkGenesis failed")
		}
	case params.ForkAltair:
		s, err = v2.InitializeFromSSZBytes(marshaled)
		if err != nil {
			return nil, errors.Wrap(err, "InitializeFromSSZBytes for ForkAltair failed")
		}
	case params.ForkMerge:
		s, err = v3.InitializeFromSSZBytes(marshaled)
		if err != nil {
			return nil, errors.Wrap(err, "InitializeFromSSZBytes for ForkMerge failed")
		}
	default:
		return nil, fmt.Errorf("unable to initialize BeaconState for fork version=%s", cf.Fork.String())
	}
	return s, nil
}

func BeaconState(marshaled []byte) (state.BeaconState, error) {
	cf, err := ConfigForkForState(marshaled)
	if err != nil {
		return nil, errors.Wrap(err, "in sniff.BeaconState error from ConfigForkForState")
	}
	var s state.BeaconState
	switch cf.Fork {
	case params.ForkGenesis:
		s, err = v1.InitializeFromSSZBytes(marshaled)
		if err != nil {
			return nil, errors.Wrap(err, "InitializeFromSSZBytes for ForkGenesis failed")
		}
	case params.ForkAltair:
		s, err = v2.InitializeFromSSZBytes(marshaled)
		if err != nil {
			return nil, errors.Wrap(err, "InitializeFromSSZBytes for ForkAltair failed")
		}
	case params.ForkMerge:
		s, err = v3.InitializeFromSSZBytes(marshaled)
		if err != nil {
			return nil, errors.Wrap(err, "InitializeFromSSZBytes for ForkMerge failed")
		}
	default:
		return nil, fmt.Errorf("unable to initialize BeaconState for fork version=%s", cf.Fork.String())
	}
	return s, nil
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
