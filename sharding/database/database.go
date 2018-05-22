package database

import (
	"path/filepath"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/micro/cli"
)

// CreateShardDB initializes a shardDB that writes to local disk.
func CreateShardDB(ctx *cli.Context, name string) (sharding.ShardBackend, error) {

	dataDir := ctx.GlobalString(utils.DataDir.Name)
	path := filepath.Join(dataDir, name)

	// Uses default cache and handles values.
	// TODO: fix interface.
	return ethdb.NewLDBDatabase(path, 16, 16)
}
