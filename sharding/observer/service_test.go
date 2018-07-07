package observer

import (
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/prysmaticlabs/geth-sharding/sharding/types"
	"github.com/prysmaticlabs/geth-sharding/sharding/database"
	"github.com/prysmaticlabs/geth-sharding/sharding/internal"
	"github.com/prysmaticlabs/geth-sharding/sharding/mainchain"
	"github.com/prysmaticlabs/geth-sharding/sharding/p2p"
	"github.com/prysmaticlabs/geth-sharding/sharding/params"
	"github.com/prysmaticlabs/geth-sharding/sharding/syncer"
)

// Verifies that Observer implements the Actor interface.
var _ = types.Actor(&Observer{})

func TestStartStop(t *testing.T) {
	h := internal.NewLogHandler(t)
	log.Root().SetHandler(h)

	server, err := p2p.NewServer()
	if err != nil {
		t.Fatalf("Unable to setup p2p server: %v", err)
	}
	shardChainDB, err := database.NewShardDB("", "", true)
	if err != nil {
		t.Fatalf("Unable to setup db: %v", err)
	}
	shardID := 0
	client := &mainchain.SMCClient{}

	syncer, err := syncer.NewSyncer(params.DefaultConfig, client, server, shardChainDB, shardID)
	if err != nil {
		t.Fatalf("Unable to setup sync service: %v", err)
	}

	observer, err := NewObserver(server, shardChainDB, shardID, syncer, client)
	if err != nil {
		t.Fatalf("Unable to set up observer service: %v", err)
	}

	observer.sync.Start()
	h.VerifyLogMsg("Starting sync service")

	observer.Start()
	h.VerifyLogMsg("Starting observer service")

	err = observer.Stop()
	if err != nil {
		t.Fatalf("Unable to stop observer service: %v", err)
	}

	h.VerifyLogMsg("Stopping observer service")

	if observer.ctx.Err() == nil {
		t.Errorf("Context was not cancelled")
	}
}
