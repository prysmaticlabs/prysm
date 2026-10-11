package light_client

import (
	"fmt"

	fieldparams "github.com/OffchainLabs/prysm/v7/config/fieldparams"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/encoding/bytesutil"
	pb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/time/slots"
	"google.golang.org/protobuf/proto"
)

// EpochData is one epoch of light client backfill data.
type EpochData interface {
	proto.Message
	GetEpoch() primitives.Epoch
	MarshalSSZ() ([]byte, error)
	UnmarshalSSZ([]byte) error
}

// EpochDataVersion returns the fork of the last non-empty block_data entry, or of the epoch's
// start slot when the window is empty. The epoch alone is not enough: a window can straddle a fork.
func EpochDataVersion(d EpochData) (int, error) {
	slot, err := epochDataForkSlot(d)
	if err != nil {
		return 0, err
	}
	return slots.ToForkVersion(slot), nil
}

// epochDataForkSlot returns the slot whose fork decides the fork of d.
func epochDataForkSlot(d EpochData) (primitives.Slot, error) {
	var last int
	var found bool
	var n int
	switch t := d.(type) {
	case *pb.LightClientEpochDataAltair:
		n, last, found = lastNonEmptyBlockData(t.GetBlockData())
	case *pb.LightClientEpochDataCapella:
		n, last, found = lastNonEmptyBlockData(t.GetBlockData())
	case *pb.LightClientEpochDataDeneb:
		n, last, found = lastNonEmptyBlockData(t.GetBlockData())
	case *pb.LightClientEpochDataElectra:
		n, last, found = lastNonEmptyBlockData(t.GetBlockData())
	case *pb.LightClientEpochDataGloas:
		n, last, found = lastNonEmptyBlockData(t.GetBlockData())
	default:
		return 0, fmt.Errorf("unsupported light client epoch data type %T", d)
	}
	if n != fieldparams.SlotsPerEpoch {
		return 0, fmt.Errorf("light client epoch data has %d block_data entries, want %d", n, fieldparams.SlotsPerEpoch)
	}

	start, err := slots.EpochStart(d.GetEpoch())
	if err != nil {
		return 0, err
	}
	if !found {
		return start, nil
	}
	// The window covers the SLOTS_PER_EPOCH slots ending at the epoch's start
	// slot, so block_data[i] describes slot start - SLOTS_PER_EPOCH + 1 + i.
	offset := primitives.Slot(fieldparams.SlotsPerEpoch - 1 - last)
	if start < offset {
		return 0, fmt.Errorf("light client epoch data for epoch %d has a block before genesis at block_data[%d]", d.GetEpoch(), last)
	}
	return start - offset, nil
}

func lastNonEmptyBlockData[T interface{ GetStateRoot() []byte }](bd []T) (int, int, bool) {
	for i := len(bd) - 1; i >= 0; i-- {
		if !bytesutil.ZeroRoot(bd[i].GetStateRoot()) {
			return len(bd), i, true
		}
	}
	return len(bd), -1, false
}
