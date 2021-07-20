package validator

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/utils"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync"
)

// Server defines a server implementation of the gRPC Validator service,
// providing RPC endpoints intended for validator clients.
type Server struct {
	HeadFetcher   blockchain.HeadFetcher
	TimeFetcher   blockchain.TimeFetcher
	SyncChecker   sync.Checker
	BlockProducer utils.BlockProducer
}
