package syncer

import (
	"testing"

	"github.com/prysmaticlabs/prysm/client/mainchain"
	"github.com/prysmaticlabs/prysm/client/params"
	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/database"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

var _ = shared.Service(&Syncer{})

func TestStop(t *testing.T) {
	hook := logTest.NewGlobal()

	config := &database.DBConfig{Name: "", DataDir: "", InMemory: true}
	shardChainDB, err := database.NewDB(config)
	if err != nil {
		t.Fatalf("unable to setup db: %v", err)
	}
	shardID := 0
	server, err := p2p.NewServer()
	if err != nil {
		t.Fatalf("Unable to setup p2p server: %v", err)
	}

	syncer, err := NewSyncer(params.DefaultConfig(), &mainchain.SMCClient{}, server, shardChainDB, shardID)
	if err != nil {
		t.Fatalf("Unable to setup sync service: %v", err)
	}

	syncer.collationBodyBuf = make(chan p2p.Message)

	if err := syncer.Stop(); err != nil {
		t.Fatalf("Unable to stop sync service: %v", err)
	}

	msg := hook.LastEntry().Message
	want := "Stopping sync service"
	if msg != want {
		t.Errorf("incorrect log, expected %s, got %s", want, msg)
	}

	// The context should have been canceled.
	if syncer.ctx.Err() == nil {
		t.Error("Context was not canceled")
	}
	hook.Reset()
}
