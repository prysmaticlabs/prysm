package eth

import (
	"bytes"
	"fmt"
	"math/bits"

	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"

	v1 "github.com/prysmaticlabs/prysm/v5/proto/eth/v1"
)

const (
	NextSyncCommitteeIndex = uint64(55)
	FinalizedRootIndex     = uint64(105)
)

func (x *SyncCommittee) Equals(other *SyncCommittee) bool {
	if len(x.Pubkeys) != len(other.Pubkeys) {
		return false
	}
	for i := range x.Pubkeys {
		if !bytes.Equal(x.Pubkeys[i], other.Pubkeys[i]) {
			return false
		}
	}
	return bytes.Equal(x.AggregatePubkey, other.AggregatePubkey)
}

func FloorLog2(x uint64) int {
	return bits.Len64(x - 1)
}

func isEmptyWithLength(bb [][]byte, length uint64) bool {
	if len(bb) == 0 {
		return true
	}
	l := FloorLog2(length)
	if len(bb) != l {
		return false
	}
	for _, b := range bb {
		if !bytes.Equal(b, []byte{}) {
			return false
		}
	}
	return true
}

func (x *LightClientUpdate) IsSyncCommitteeUpdate() bool {
	return !isEmptyWithLength(x.GetNextSyncCommitteeBranch(), NextSyncCommitteeIndex)
}

func (x *LightClientUpdate) IsFinalityUpdate() bool {
	return !isEmptyWithLength(x.GetFinalityBranch(), FinalizedRootIndex)
}

func (x *LightClientHeaderContainer) GetBeacon() (*v1.BeaconBlockHeader, error) {
	switch input := x.Header.(type) {
	case *LightClientHeaderContainer_HeaderAltair:
		return input.HeaderAltair.Beacon, nil
	case *LightClientHeaderContainer_HeaderCapella:
		return input.HeaderCapella.Beacon, nil
	case *LightClientHeaderContainer_HeaderDeneb:
		return input.HeaderDeneb.Beacon, nil
	default:
		return nil, fmt.Errorf("unknown header type: %T", input)
	}
}

func (x *LightClientHeaderContainer) GetExecutionHeaderCapella() (*enginev1.ExecutionPayloadHeaderCapella, error) {
	switch input := x.Header.(type) {
	case *LightClientHeaderContainer_HeaderCapella:
		return input.HeaderCapella.Execution, nil
	default:
		return nil, fmt.Errorf("header type %T not Capella", input)
	}
}

func (x *LightClientHeaderContainer) GetExecutionHeaderDeneb() (*enginev1.ExecutionPayloadHeaderDeneb, error) {
	switch input := x.Header.(type) {
	case *LightClientHeaderContainer_HeaderDeneb:
		return input.HeaderDeneb.Execution, nil
	default:
		return nil, fmt.Errorf("header type %T not Deneb", input)
	}
}

func (x *LightClientHeaderContainer) GetExecutionBranch() ([][]byte, error) {
	switch input := x.Header.(type) {
	case *LightClientHeaderContainer_HeaderCapella:
		return input.HeaderCapella.ExecutionBranch, nil
	case *LightClientHeaderContainer_HeaderDeneb:
		return input.HeaderDeneb.ExecutionBranch, nil
	default:
		return nil, fmt.Errorf("wrong header type %T", input)
	}
}
