package sniff

import (
	"encoding/binary"
	"fmt"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
)

type fieldType int

const (
	TypeUint64 fieldType = iota
	TypeBytes4
	TypeByteSlice
	TypeRoot
	TypeContainer
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

type ConfigFork struct {
	Config params.ConfigName
	Fork   params.ForkName
	Epoch  types.Epoch
}

func ConfigForkForState(marshaled []byte) (*ConfigFork, error) {
	epoch, err := beaconStateEpoch.Uint64(marshaled)
	if err != nil {
		return nil, err
	}
	cv, err := beaconStateCurrentVersion.Bytes4(marshaled)
	if err != nil {
		return nil, err
	}
	return FindConfigFork(types.Epoch(epoch), cv)
}

func FindConfigFork(epoch types.Epoch, cv [4]byte) (*ConfigFork, error) {
	cf := &ConfigFork{
		Epoch: epoch,
	}
	for name, cfg := range params.AllConfigs() {
		genesis := bytesutil.ToBytes4(cfg.GenesisForkVersion)
		altair := bytesutil.ToBytes4(cfg.AltairForkVersion)
		merge := bytesutil.ToBytes4(cfg.MergeForkVersion)
		for id := range cfg.ForkVersionSchedule {
			if id == cv {
				cf.Config = name
				switch id {
				case genesis:
					cf.Fork = params.ForkGenesis
				case altair:
					cf.Fork = params.ForkAltair
				case merge:
					cf.Fork = params.ForkMerge
				default:
					return cf, fmt.Errorf("unrecognized fork for config name=%s, BeaconState.fork.current_version=%#x", name.String(), cv)
				}
				return cf, nil
			}
		}
	}
	return cf, fmt.Errorf("could not find a config+fork match for epoch=%d, current_version=%#x", epoch, cv)
}

var beaconBlockSlot = fieldSpec{
	// ssz variable length offset (not to be confused with the fieldSpec offest) is a uint32
	// variable length offsets come before fixed length data, so that's 4 bytes at the beginning
	// then signature is 96 bytes, 4+96 = 100
	offset: 100,
	size:   8,
	t:      TypeUint64,
}

func SlotFromBlock(marshaled []byte) (types.Slot, error) {
	slot, err := beaconBlockSlot.Uint64(marshaled)
	if err != nil {
		return 0, err
	}
	return types.Slot(slot), nil
}
