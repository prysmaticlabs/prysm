package ssz

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/urfave/cli/v2"
)

var sniffFlags = struct {
	Path string
}{}

var inspectCmd = &cli.Command{
	Name:   "sniff",
	Usage:  "Extract state metadata from a serialized ssz file",
	Action: sniff,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "state-path",
			Usage:       "path to file with ssz-serialized state root.`",
			Destination: &sniffFlags.Path,
		},
	},
}

/*
class BeaconState(Container):
    # Versioning
    genesis_time: uint64
    genesis_validators_root: Root // [32]byte
    slot: Slot // uint64
    fork: Fork // [16]byte, nested

class Fork(Container):
    previous_version: Version 			 // [4]byte
    current_version: Version             // [4]byte
    epoch: Epoch  # Epoch of latest fork // uint64
*/

type fieldType int

const (
	TypeUint64 fieldType = iota
	TypeByteSlice
	TypeRoot
	TypeContainer
)

type field struct {
	name      string
	size      int
	nested    []*field
	fieldType fieldType
	slice     []byte
}

var forkFields = []*field{
	{size: 4, fieldType: TypeByteSlice, name: "previous_version"},
	{size: 4, fieldType: TypeByteSlice, name: "current_version"},
	{size: 8, fieldType: TypeUint64, name: "epoch"},
}

var beaconState = &field{fieldType: TypeContainer, name: "BeaconState", nested: []*field{
	{size: 8, fieldType: TypeUint64, name: "genesis_time"},
	{size: 32, fieldType: TypeRoot, name: "genesis_validators_root"},
	{size: 8, fieldType: TypeUint64, name: "slot"},
	{fieldType: TypeContainer, name: "fork", nested: forkFields},
}}

func (f *field) ByteSlice() []byte {
	if f.fieldType != TypeByteSlice {
		panic(fmt.Sprintf("ByteSlice called on non-byte slice field: %v", f))
	}
	return f.slice
}

func (f *field) Root() [32]byte {
	if f.fieldType != TypeRoot {
		panic(fmt.Sprintf("Root called on non-byte slice field: %v", f))
	}
	var r [32]byte
	copy(r[:], f.slice)
	return r
}

func (f *field) Uint64() uint64 {
	if f.fieldType != TypeUint64 {
		panic(fmt.Sprintf("Uint64 called on non-uint64 field: %v", f))
	}
	if len(f.slice) != 8 {
		panic(fmt.Sprintf("Invalid data (len of .slice s %d, expected 8), for uint64 field: %v", len(f.slice), f))
	}
	return binary.LittleEndian.Uint64(f.slice[0:8])
}

func sniff(c *cli.Context) error {
	fh, err := os.Open(sniffFlags.Path)
	if err != nil {
		return err
	}
	buf := make([]byte, beaconState.Size())
	_, err = io.ReadFull(fh, buf)
	if err != nil {
		return err
	}
	if err := beaconState.index(buf); err != nil {
		return err
	}
	dumpField(beaconState)
	cf, err := ConfigForkFromBeaconState(beaconState)
	if err != nil {
		return err
	}
	fmt.Printf("\nfork metadata:\n%s\n", cf.String())
	return nil
}

func dumpField(f *field, indent ...string) {
	for _, s := range indent {
		fmt.Print(s)
	}
	switch f.fieldType {
	case TypeContainer:
		fmt.Printf("%s(container):\n", f.name)
		for _, n := range f.nested {
			dumpField(n, append(indent, "\t")...)
		}
	case TypeByteSlice:
		fmt.Printf("%s(bytes): %#x\n", f.name, f.ByteSlice())
	case TypeRoot:
		fmt.Printf("%s(root): %#x\n", f.name, f.Root())
	case TypeUint64:
		fmt.Printf("%s(uint64): %d\n", f.name, f.Uint64())
	default:
		panic(fmt.Sprintf("unhandled field type for field=%v", f))
	}
}

func (f *field) Size() int {
	if len(f.nested) > 0 {
		size := 0
		for _, n := range f.nested {
			size += n.Size()
		}
		return size
	}
	return f.size
}

func (f *field) index(buf []byte) error {
	// always copy the buf to slice; containers will have a slice to their contents, primitives to their values
	f.slice = buf
	// if the field is a primitive, stop here
	if len(f.nested) == 0 {
		return nil
	}

	// getting this far means we're looking at a container of some kind, recursively index it
	// we're always working with a sliced view, so index math is simple
	offset := 0
	for _, fld := range f.nested {
		if err := fld.index(buf[offset : offset+fld.Size()]); err != nil {
			return err
		}
		offset += fld.Size()
	}
	return nil
}

type path string

func (p path) head() path {
	parts := strings.SplitN(string(p), ".", 2)
	return path(parts[0])
}

func (p path) tail() path {
	parts := strings.SplitN(string(p), ".", 2)
	if len(parts) == 2 {
		return path(parts[1])
	}
	return ""
}

func (p path) leaf() bool {
	return !strings.Contains(string(p), ".")
}

func (f *field) get(p path) (*field, error) {
	name := string(p.head())
	if f.name != name {
		return nil, fmt.Errorf("could not match path = %s", p)
	}
	if p.leaf() {
		return f, nil
	}
	childName := string(p.tail().head())
	for _, n := range f.nested {
		if n.name != childName {
			continue
		}
		return n.get(p.tail())
	}
	return nil, fmt.Errorf("could not match path = %s", p.tail())
}

type ConfigFork struct {
	Config params.ConfigName
	Fork   params.ForkName
	Epoch  types.Epoch
}

func (cf ConfigFork) String() string {
	return fmt.Sprintf("config=%s, fork=%s, epoch=%d", cf.Config.String(), cf.Fork.String(), cf.Epoch)
}

func ConfigForkFromBeaconState(f *field) (ConfigFork, error) {
	cf := ConfigFork{}
	currentVersion, err := f.get("BeaconState.fork.current_version")
	if err != nil {
		return cf, err
	}
	var cv [4]byte
	copy(cv[:], currentVersion.ByteSlice())
	e, err := f.get("BeaconState.fork.epoch")
	if err != nil {
		return cf, err
	}
	cf.Epoch = types.Epoch(e.Uint64())
	for name, cfg := range params.AllConfigs() {
		genesis := tobyte4(cfg.GenesisForkVersion)
		altair := tobyte4(cfg.AltairForkVersion)
		merge := tobyte4(cfg.MergeForkVersion)
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
	return cf, fmt.Errorf("Could not find a match for BeaconState.fork.current_version=%#x", cv)
}

func tobyte4(slice []byte) [4]byte {
	var b4 [4]byte
	copy(b4[:], slice)
	return b4
}
