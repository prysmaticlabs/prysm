package v1

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethrpc "github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	"google.golang.org/protobuf/types/known/timestamppb"
	"strconv"
	"time"
)

func NewSyncAggregateFromJSON(syncAggregate *ethrpc.SyncAggregateJson) *SyncAggregate {
	return &SyncAggregate{
		SyncCommitteeBits:      hexutil.MustDecode(syncAggregate.SyncCommitteeBits),
		SyncCommitteeSignature: hexutil.MustDecode(syncAggregate.SyncCommitteeSignature),
	}
}

func timeFromJSON(timestamp string) (*time.Time, error) {
	timeInt, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return nil, err
	}
	t := time.Unix(timeInt, 0)
	return &t, nil
}

func NewGenesisResponse_GenesisFromJSON(genesis *ethrpc.GenesisResponse_GenesisJson) (
	*GenesisResponse_Genesis, error) {
	genesisTime, err := timeFromJSON(genesis.GenesisTime)
	if err != nil {
		return nil, err
	}
	return &GenesisResponse_Genesis{
		GenesisTime:           timestamppb.New(*genesisTime),
		GenesisValidatorsRoot: hexutil.MustDecode(genesis.GenesisValidatorsRoot),
		GenesisForkVersion:    hexutil.MustDecode(genesis.GenesisForkVersion),
	}, nil
}
