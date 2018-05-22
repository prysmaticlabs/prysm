package database

import (
	"path/filepath"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/micro/cli"
)

// NewShardDB initializes a shardDB that writes to local disk.
func NewShardDB(ctx *cli.Context, name string) (sharding.ShardBackend, error) {

	dataDir := ctx.GlobalString(utils.DataDirFlag.Name)
	path := filepath.Join(dataDir, name)

	// Uses default cache and handles values.
	// TODO: allow these to be set based on cli context.
	// TODO: fix interface - lots of methods do not match.
	return ethdb.NewLDBDatabase(path, 16, 16)
}
