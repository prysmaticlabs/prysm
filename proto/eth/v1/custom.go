package v1

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethrpc "github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
)

func NewSyncAggregateFromJSON(syncAggregate *ethrpc.SyncAggregateJson) *SyncAggregate {
	return &SyncAggregate{
		SyncCommitteeBits:      hexutil.MustDecode(syncAggregate.SyncCommitteeBits),
		SyncCommitteeSignature: hexutil.MustDecode(syncAggregate.SyncCommitteeSignature),
	}
}
