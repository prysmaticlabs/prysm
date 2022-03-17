package detect

import (
	"encoding/binary"
	"fmt"

	ssz "github.com/ferranbt/fastssz"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/state/v1"
	v2 "github.com/prysmaticlabs/prysm/beacon-chain/state/v2"
	v3 "github.com/prysmaticlabs/prysm/beacon-chain/state/v3"
	v1alpha1 "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/time/slots"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
)

type fieldType int

const (
	TypeUint64 fieldType = iota
	TypeBytes4
)

type fieldSpec struct {
	offset int
	size   int
	t      fieldType
}

func (f *fieldSpec) Uint64(state []byte) (uint64, error) {
	if f.t != TypeUint64 {
		return 0, fmt.Errorf("Uint64 called on non-uint64 field: %v", f)
	}
	s, err := f.slice(state)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(s), nil
}

func (f *fieldSpec) Bytes4(state []byte) ([4]byte, error) {
	var b4 [4]byte
	if f.t != TypeBytes4 {
		return b4, fmt.Errorf("Bytes4 called on non-bytes4 field %v", f)
	}
	if f.size != 4 {
		return b4, fmt.Errorf("Bytes4 types must have a size of 4, invalid fieldSpec %v", f)
	}
	val, err := f.slice(state)
	if err != nil {
		return b4, err
	}
	return bytesutil.ToBytes4(val), nil
}

func (f *fieldSpec) slice(value []byte) ([]byte, error) {
	if len(value) < f.offset+f.size {
		return nil, fmt.Errorf("cannot pull bytes from value; offset=%d, size=%d, so value must be at least %d bytes (actual=%d)", f.offset, f.size, f.offset+f.size, len(value))
	}
	return value[f.offset : f.offset+f.size], nil
}

// ConfigFork represents the intersection of Configuration (eg mainnet, testnet) and Fork (eg phase0, altair).
// Using a detected ConfigFork, a BeaconState or SignedBeaconBlock can be correctly unmarshaled without the need to
// hard code a concrete type in paths where only the marshaled bytes, or marshaled bytes and a version, are available.
type ConfigFork struct {
	ConfigName params.ConfigName
	Config     *params.BeaconChainConfig
	Fork       params.ForkName
	Version    [4]byte
	Epoch      types.Epoch
}

// ByVersion uses a lookup table to resolve a Version (from a beacon node api for instance, or obtained by peeking at
// the bytes of a marshaled BeaconState) to a ConfigFork.
func ByVersion(cv [4]byte) (*ConfigFork, error) {
	cf := &ConfigFork{
		Version: cv,
	}
	for name, cfg := range params.AllConfigs() {
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
					cf.Fork = params.ForkGenesis
				case altair:
					cf.Fork = params.ForkAltair
				case merge:
					cf.Fork = params.ForkBellatrix
				default:
					return cf, fmt.Errorf("unrecognized fork for config name=%s, BeaconState.fork.current_version=%#x", name.String(), cv)
				}
				return cf, nil
			}
		}
	}
	return cf, fmt.Errorf("could not find a config+fork match with version=%#x", cv)
}

var beaconStateCurrentVersion = fieldSpec{
	// 52 = 8 (genesis_time) + 32 (genesis_validators_root) + 8 (slot) + 4 (previous_version)
	offset: 52,
	size:   4,
	t:      TypeBytes4,
}

var beaconStateEpoch = fieldSpec{
	// 52 = 8 (genesis_time) + 32 (genesis_validators_root) + 8 (slot) + 4 (previous_version) + 4 (current_version)
	offset: 56,
	size:   8,
	t:      TypeUint64,
}

func currentVersionFromState(marshaled []byte) ([4]byte, error) {
	return beaconStateCurrentVersion.Bytes4(marshaled)
}

// ByState exploits the fixed-size lower-order bytes in a BeaconState as a heuristic to obtain the value of the
// state.version field without first unmarshaling the BeaconState. The Version is then internally used to lookup
// the correct ConfigVersion.
func ByState(marshaled []byte) (*ConfigFork, error) {
	cv, err := currentVersionFromState(marshaled)
	if err != nil {
		return nil, err
	}
	return ByVersion(cv)
}

// UnmarshalBeaconState uses internal knowledge in the ConfigFork to pick the right concrete BeaconState type,
// then Unmarshal()s the type and returns an instance of state.BeaconState if successful.
func (cf *ConfigFork) UnmarshalBeaconState(marshaled []byte) (s state.BeaconState, err error) {
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
	case params.ForkBellatrix:
		s, err = v3.InitializeFromSSZBytes(marshaled)
		if err != nil {
			return nil, errors.Wrap(err, "InitializeFromSSZBytes for ForkMerge failed")
		}
	default:
		return nil, fmt.Errorf("unable to initialize BeaconState for fork version=%s", cf.Fork.String())
	}
	return s, nil
}

var beaconBlockSlot = fieldSpec{
	// ssz variable length offset (not to be confused with the fieldSpec offest) is a uint32
	// variable length. Offsets come before fixed length data, so that's 4 bytes at the beginning
	// then signature is 96 bytes, 4+96 = 100
	offset: 100,
	size:   8,
	t:      TypeUint64,
}

func slotFromBlock(marshaled []byte) (types.Slot, error) {
	slot, err := beaconBlockSlot.Uint64(marshaled)
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
	case params.ForkGenesis:
		blk = &v1alpha1.SignedBeaconBlock{}
	case params.ForkAltair:
		blk = &v1alpha1.SignedBeaconBlockAltair{}
	case params.ForkBellatrix:
		blk = &v1alpha1.SignedBeaconBlockBellatrix{}
	default:
		return nil, fmt.Errorf("unable to initialize BeaconBlock for fork version=%s at slot=%d", cf.Fork.String(), slot)
	}
	err = blk.UnmarshalSSZ(marshaled)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal SignedBeaconBlock in BlockFromSSZReader")
	}
	return wrapper.WrappedSignedBeaconBlock(blk)
}
