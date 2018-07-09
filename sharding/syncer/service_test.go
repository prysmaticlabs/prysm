package syncer

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/prysmaticlabs/geth-sharding/sharding/mainchain"
	"github.com/prysmaticlabs/geth-sharding/sharding/params"

	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/prysmaticlabs/geth-sharding/sharding/p2p/messages"

	logTest "github.com/sirupsen/logrus/hooks/test"

	"github.com/prysmaticlabs/geth-sharding/sharding/database"
	"github.com/prysmaticlabs/geth-sharding/sharding/p2p"
	"github.com/prysmaticlabs/geth-sharding/sharding/types"
)

var _ = types.Service(&Syncer{})

func TestStop(t *testing.T) {
	hook := logTest.NewGlobal()

	shardChainDB, err := database.NewShardDB("", "", true)
	if err != nil {
		t.Fatalf("unable to setup db: %v", err)
	}
	shardID := 0
	server, err := p2p.NewServer()
	if err != nil {
		t.Fatalf("Unable to setup p2p server: %v", err)
	}

	syncer, err := NewSyncer(params.DefaultConfig, &mainchain.SMCClient{}, server, shardChainDB, shardID)
	if err != nil {
		t.Fatalf("Unable to setup sync service: %v", err)
	}

	feed := server.Feed(messages.CollationBodyRequest{})
	syncer.msgChan = make(chan p2p.Message)
	syncer.errChan = make(chan error)
	syncer.bodyRequests = feed.Subscribe(syncer.msgChan)

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
}

// This test uses a faulty Signer interface in order to trigger an error
// in the simulateNotaryRequests goroutine when attempting to sign
// a collation header within the goroutine's internals.
func TestHandleCollationBodyRequests_FaultySigner(t *testing.T) {
	shardChainDB, err := database.NewShardDB("", "", true)
	if err != nil {
		t.Fatalf("unable to setup db: %v", err)
	}
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
	shard := types.NewShard(big.NewInt(int64(shardID)), shardChainDB.DB())

	syncer.msgChan = make(chan p2p.Message)
	syncer.errChan = make(chan error)
	syncer.bodyRequests = feed.Subscribe(syncer.msgChan)

	go syncer.HandleCollationBodyRequests(shard)

	msg := p2p.Message{
		Peer: p2p.Peer{},
		Data: messages.CollationBodyRequest{},
	}
	syncer.msgChan <- msg
	receivedErr := <-syncer.errChan
	expectedErr := "could not construct response"
	if !strings.Contains(receivedErr.Error(), expectedErr) {
		t.Errorf("Expected error did not match. want: %v, got: %v", expectedErr, receivedErr)
	}

	syncer.cancel()

	// The context should have been canceled.
	if syncer.ctx.Err() == nil {
		t.Error("Context was not canceled")
	}
}

// This test checks the proper functioning of the handleCollationBodyRequests goroutine
// by listening to the responseSent channel which occurs after successful
// construction and sending of a response via p2p.
func TestHandleCollationBodyRequests(t *testing.T) {
	hook := logTest.NewGlobal()

	shardChainDB, err := database.NewShardDB("", "", true)
	if err != nil {
		t.Fatalf("unable to setup db: %v", err)
	}
	server, err := p2p.NewServer()
	if err != nil {
		t.Fatalf("Unable to setup p2p server: %v", err)
	}

	body := []byte{1, 2, 3, 4, 5}
	shardID := big.NewInt(0)
	chunkRoot := gethTypes.DeriveSha(types.Chunks(body))
	period := big.NewInt(0)
	proposerAddress := common.BytesToAddress([]byte{})

	header := types.NewCollationHeader(shardID, &chunkRoot, period, &proposerAddress, [32]byte{})

	// Stores the collation into the inmemory kv store shardChainDB.
	collation := types.NewCollation(header, body, nil)

	shard := types.NewShard(shardID, shardChainDB.DB())

	if err := shard.SaveCollation(collation); err != nil {
		t.Fatalf("Could not store collation in shardChainDB: %v", err)
	}

	syncer, err := NewSyncer(params.DefaultConfig, &mainchain.SMCClient{}, server, shardChainDB, 0)
	if err != nil {
		t.Fatalf("Unable to setup syncer service: %v", err)
	}

	feed := server.Feed(messages.CollationBodyRequest{})

	syncer.msgChan = make(chan p2p.Message)
	syncer.errChan = make(chan error)
	syncer.bodyRequests = feed.Subscribe(syncer.msgChan)

	go syncer.HandleCollationBodyRequests(shard)

	msg := p2p.Message{
		Peer: p2p.Peer{},
		Data: messages.CollationBodyRequest{
			ChunkRoot: &chunkRoot,
			ShardID:   shardID,
			Period:    period,
			Proposer:  &proposerAddress,
		},
	}
	syncer.msgChan <- msg

	logMsg := hook.AllEntries()[0].Message
	want := fmt.Sprintf("Received p2p request of type: %T", p2p.Message{})
	if logMsg != want {
		t.Errorf("incorrect log, expected %s, got %s", want, logMsg)
	}

	logMsg = hook.AllEntries()[1].Message
	want = fmt.Sprintf("Responding to p2p request with collation with headerHash: %v", header.Hash().Hex())
	if logMsg != want {
		t.Errorf("incorrect log, expected %s, got %s", want, logMsg)
	}

	syncer.cancel()
	// The context should have been canceled.
	if syncer.ctx.Err() == nil {
		t.Error("Context was not canceled")
	}
}
