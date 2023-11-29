package blob

import (
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/db/filesystem"
)

type Server struct {
	ChainInfoFetcher blockchain.ChainInfoFetcher
	BeaconDB         db.ReadOnlyDatabase
	BlobStorage      *filesystem.BlobStorage
}
