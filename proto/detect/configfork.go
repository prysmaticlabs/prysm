package detect

import (
	"fmt"

	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/runtime/version"

	ssz "github.com/ferranbt/fastssz"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/state/v1"
	v2 "github.com/prysmaticlabs/prysm/beacon-chain/state/v2"
	v3 "github.com/prysmaticlabs/prysm/beacon-chain/state/v3"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/time/slots"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
)

// ConfigFork represents the intersection of Configuration (eg mainnet, testnet) and Fork (eg phase0, altair).
// Using a detected ConfigFork, a BeaconState or SignedBeaconBlock can be correctly unmarshaled without the need to
// hard code a concrete type in paths where only the marshaled bytes, or marshaled bytes and a version, are available.
type ConfigFork struct {
	ConfigName params.ConfigName
	Config     *params.BeaconChainConfig
	// Fork aligns with the fork names in config/params/values.go
	Fork    int
	Version [fieldparams.VersionLength]byte
	Epoch   types.Epoch
}

var beaconStateCurrentVersion = fieldSpec{
	// 52 = 8 (genesis_time) + 32 (genesis_validators_root) + 8 (slot) + 4 (previous_version)
	offset: 52,
	t:      typeBytes4,
}

// ByState exploits the fixed-size lower-order bytes in a BeaconState as a heuristic to obtain the value of the
// state.version field without first unmarshaling the BeaconState. The Version is then internally used to lookup
// the correct ConfigVersion.
func ByState(marshaled []byte) (*ConfigFork, error) {
	cv, err := beaconStateCurrentVersion.bytes4(marshaled)
	if err != nil {
		return nil, err
	}
	return ByVersion(cv)
}

// ByVersion uses a lookup table to resolve a Version (from a beacon node api for instance, or obtained by peeking at
// the bytes of a marshaled BeaconState) to a ConfigFork.
func ByVersion(cv [fieldparams.VersionLength]byte) (*ConfigFork, error) {
	cf := &ConfigFork{
		Version: cv,
	}
	for name, cfg := range params.AllConfigs {
		genesis := bytesutil.ToBytes4(cfg.GenesisForkVersion)
		altair := bytesutil.ToBytes4(cfg.AltairForkVersion)
		merge := bytesutil.ToBytes4(cfg.BellatrixForkVersion)
		for v, e := range cfg.ForkVersionSchedule {
			if v == cv {
				cf.ConfigName = name
				cf.Config = cfg
				cf.Epoch = e
				switch v {
				case genesis:
					cf.Fork = version.Phase0
				case altair:
					cf.Fork = version.Altair
				case merge:
					cf.Fork = version.Bellatrix
				default:
					return cf, fmt.Errorf("unrecognized fork for config name=%s, BeaconState.fork.current_version=%#x", name.String(), cv)
				}
				return cf, nil
			}
		}
	}
	return cf, fmt.Errorf("could not find a config+fork match with version=%#x", cv)
}

// UnmarshalBeaconState uses internal knowledge in the ConfigFork to pick the right concrete BeaconState type,
// then Unmarshal()s the type and returns an instance of state.BeaconState if successful.
func (cf *ConfigFork) UnmarshalBeaconState(marshaled []byte) (s state.BeaconState, err error) {
	forkName := version.String(cf.Fork)
	switch fork := cf.Fork; fork {
	case version.Phase0:
		st := &ethpb.BeaconState{}
		err = st.UnmarshalSSZ(marshaled)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal state, detected fork=%s", forkName)
		}
		s, err = v1.InitializeFromProtoUnsafe(st)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to init state trie from state, detected fork=%s", forkName)
		}
	case version.Altair:
		st := &ethpb.BeaconStateAltair{}
		err = st.UnmarshalSSZ(marshaled)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal state, detected fork=%s", forkName)
		}
		s, err = v2.InitializeFromProtoUnsafe(st)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to init state trie from state, detected fork=%s", forkName)
		}
	case version.Bellatrix:
		st := &ethpb.BeaconStateBellatrix{}
		err = st.UnmarshalSSZ(marshaled)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal state, detected fork=%s", forkName)
		}
		s, err = v3.InitializeFromProtoUnsafe(st)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to init state trie from state, detected fork=%s", forkName)
		}
	default:
		return nil, fmt.Errorf("unable to initialize BeaconState for fork version=%s", forkName)
	}

	return s, nil
}

var beaconBlockSlot = fieldSpec{
	// ssz variable length offset (not to be confused with the fieldSpec offest) is a uint32
	// variable length. Offsets come before fixed length data, so that's 4 bytes at the beginning
	// then signature is 96 bytes, 4+96 = 100
	offset: 100,
	t:      typeUint64,
}

func slotFromBlock(marshaled []byte) (types.Slot, error) {
	slot, err := beaconBlockSlot.uint64(marshaled)
	if err != nil {
		return 0, err
	}
	return types.Slot(slot), nil
}

// UnmarshalBeaconBlock uses internal knowledge in the ConfigFork to pick the right concrete SignedBeaconBlock type,
// then Unmarshal()s the type and returns an instance of block.SignedBeaconBlock if successful.
func (cf *ConfigFork) UnmarshalBeaconBlock(marshaled []byte) (block.SignedBeaconBlock, error) {
	slot, err := slotFromBlock(marshaled)
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

	var blk ssz.Unmarshaler
	switch cf.Fork {
	case version.Phase0:
		blk = &ethpb.SignedBeaconBlock{}
	case version.Altair:
		blk = &ethpb.SignedBeaconBlockAltair{}
	case version.Bellatrix:
		blk = &ethpb.SignedBeaconBlockBellatrix{}
	default:
		forkName := version.String(cf.Fork)
		return nil, fmt.Errorf("unable to initialize BeaconBlock for fork version=%s at slot=%d", forkName, slot)
	}
	err = blk.UnmarshalSSZ(marshaled)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal SignedBeaconBlock in BlockFromSSZReader")
	}
	return wrapper.WrappedSignedBeaconBlock(blk)
}
