package syncer

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/sharding/p2p/messages"

	"github.com/ethereum/go-ethereum/sharding/mainchain"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/database"
	internal "github.com/ethereum/go-ethereum/sharding/internal"
	"github.com/ethereum/go-ethereum/sharding/p2p"
	"github.com/ethereum/go-ethereum/sharding/params"
)

var _ = sharding.Service(&Syncer{})

func TestStartStop(t *testing.T) {
	h := internal.NewLogHandler(t)
	log.Root().SetHandler(h)

	shardChainDB := database.NewShardKV()
	shardID := 0
	server, err := p2p.NewServer()
	if err != nil {
		t.Fatalf("Unable to setup p2p server: %v", err)
	}

	syncer, err := NewSyncer(params.DefaultConfig, &mainchain.SMCClient{}, server, shardChainDB, shardID)
	if err != nil {
		t.Fatalf("Unable to setup syncer service: %v", err)
	}

	syncer.Start()
	h.VerifyLogMsg("Starting sync service")

	if err := syncer.Stop(); err != nil {
		t.Fatalf("Unable to stop syncer service: %v", err)
	}
	h.VerifyLogMsg("Stopping sync service")

	// The context should have been cancelled.
	if syncer.ctx.Err() == nil {
		t.Error("Context was not cancelled")
	}
}

func TestHandleCollationBodyRequests(t *testing.T) {
	h := internal.NewLogHandler(t)
	log.Root().SetHandler(h)

	shardChainDB := database.NewShardKV()
	shardID := 0
	server, err := p2p.NewServer()
	if err != nil {
		t.Fatalf("Unable to setup p2p server: %v", err)
	}

	syncer, err := NewSyncer(params.DefaultConfig, &mainchain.SMCClient{}, server, shardChainDB, shardID)
	if err != nil {
		t.Fatalf("Unable to setup syncer service: %v", err)
	}

	feed := server.Feed(messages.CollationBodyRequest{})

	go syncer.handleCollationBodyRequests(feed)

	msg := p2p.Message{
		Peer: p2p.Peer{},
		Data: messages.CollationBodyRequest{},
	}
	feed.Send(msg)

	h.VerifyLogMsg(fmt.Sprintf("Received p2p request of type: %T", msg))

	if err := syncer.Stop(); err != nil {
		t.Fatalf("Unable to stop syncer service: %v", err)
	}

	h.VerifyLogMsg("Stopping sync service")

	// The context should have been cancelled.
	if syncer.ctx.Err() == nil {
		t.Error("Context was not cancelled")
	}
}
