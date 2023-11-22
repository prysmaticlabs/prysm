package blob

import (
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/db"
)

type Server struct {
	ChainInfoFetcher blockchain.ChainInfoFetcher
	BeaconDB         db.ReadOnlyDatabase
}
