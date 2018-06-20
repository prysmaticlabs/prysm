package observer

import (
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding"
	internal "github.com/ethereum/go-ethereum/sharding/internal"
	"github.com/ethereum/go-ethereum/sharding/p2p"
	"github.com/ethereum/go-ethereum/sharding/database"
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
	shardChainDB := database.NewShardKV()
	shardID := 0

	observer, err := NewObserver(server, shardChainDB, shardID)
	if err != nil {
		t.Fatalf("Unable to set up observer service: %v", err)
	}

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
