package observer

import (
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/database"
	"github.com/ethereum/go-ethereum/sharding/internal"
	"github.com/ethereum/go-ethereum/sharding/mainchain"
	"github.com/ethereum/go-ethereum/sharding/p2p"
	"github.com/ethereum/go-ethereum/sharding/params"
	"github.com/ethereum/go-ethereum/sharding/syncer"
)

// Verifies that Observer implements the Actor interface.
var _ = sharding.Actor(&Observer{})

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
