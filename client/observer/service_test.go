package observer

import (
	"testing"

	"github.com/prysmaticlabs/prysm/client/database"
	"github.com/prysmaticlabs/prysm/client/mainchain"
	"github.com/prysmaticlabs/prysm/client/params"
	"github.com/prysmaticlabs/prysm/client/syncer"
	"github.com/prysmaticlabs/prysm/client/types"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

// Verifies that Observer implements the Actor interface.
var _ = types.Actor(&Observer{})

func TestStartStop(t *testing.T) {

	hook := logTest.NewGlobal()

	server, err := p2p.NewServer()
	if err != nil {
		t.Fatalf("Unable to setup p2p server: %v", err)
	}
	config := &database.ShardDBConfig{Name: "", DataDir: "", InMemory: true}
	shardChainDB, err := database.NewShardDB(config)
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
	msg := hook.LastEntry().Message
	if msg != "Starting sync service" {
		t.Errorf("incorrect log, expected %s, got %s", "Starting sync service", msg)
	}

	observer.Start()
	msg = hook.LastEntry().Message
	if msg != "Starting observer service" {
		t.Errorf("incorrect log, expected %s, got %s", "Starting observer service", msg)
	}

	err = observer.Stop()
	if err != nil {
		t.Fatalf("Unable to stop observer service: %v", err)
	}

	msg = hook.LastEntry().Message
	if msg != "Stopping observer service" {
		t.Errorf("incorrect log, expected %s, got %s", "Stopping observer service", msg)
	}

	if observer.ctx.Err() == nil {
		t.Errorf("Context was not cancelled")
	}
}
