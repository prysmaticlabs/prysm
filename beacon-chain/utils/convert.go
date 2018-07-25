package utils

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	types "github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/sharding/v1"
	"golang.org/x/crypto/blake2s"
)

// ConvertToBeaconBlock provides a helpful utility to convert a protobuf BeaconBlockResponse
// to a proper type.
func ConvertToBeaconBlock(data *pb.BeaconBlockResponse) (*types.Block, error) {
	parentHash, err := blake2s.New256(data.ParentHash)
	if err != nil {
		return nil, err
	}
	randaoReveal, err := blake2s.New256(data.RandaoReveal)
	if err != nil {
		return nil, err
	}
	activeStateHash, err := blake2s.New256(data.ActiveStateHash)
	if err != nil {
		return nil, err
	}
	crystallizedStateHash, err := blake2s.New256(data.CrystallizedStateHash)
	if err != nil {
		return nil, err
	}
	blockData := &types.Data{
		ParentHash:            parentHash,
		SlotNumber:            data.SlotNumber,
		RandaoReveal:          randaoReveal,
		MainChainRef:          common.BytesToHash(data.MainChainRef),
		ActiveStateHash:       activeStateHash,
		CrystallizedStateHash: crystallizedStateHash,
		// TODO: Handle this appropriately.
		Timestamp: time.Now(),
	}
	return types.NewBlockWithData(blockData), nil
}
