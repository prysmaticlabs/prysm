package syncer

import (
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum/sharding/p2p/messages"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/database"
	internal "github.com/ethereum/go-ethereum/sharding/internal"
	"github.com/ethereum/go-ethereum/sharding/p2p"
)

var _ = sharding.Service(&Syncer{})

// This test uses a faulty Signer interface in order to trigger an error
// in the simulateNotaryRequests goroutine when attempting to sign
// a collation header within the goroutine's internals.
func TestHandleCollationBodyRequests_FaultySigner(t *testing.T) {
	h := internal.NewLogHandler(t)
	log.Root().SetHandler(h)

	shardChainDB := database.NewShardKV()
	shardID := big.NewInt(0)
	server, err := p2p.NewServer()
	if err != nil {
		t.Fatalf("Unable to setup p2p server: %v", err)
	}

	msgChan := make(chan p2p.Message, 1)
	feed := server.Feed(messages.CollationBodyRequest{})
	shard := sharding.NewShard(shardID, shardChainDB)
	done := make(chan struct{})
	errChan := make(chan error)
	sub := feed.Subscribe(msgChan)

	go handleCollationBodyRequests(&faultySigner{}, shard, server, msgChan, done, sub, errChan)

	msg := p2p.Message{
		Peer: p2p.Peer{},
		Data: messages.CollationBodyRequest{},
	}
	msgChan <- msg
	receivedErr := <-errChan
	expectedErr := "could not construct response"
	if !strings.Contains(receivedErr.Error(), expectedErr) {
		t.Errorf("Expected error did not match. want: %v, got: %v", expectedErr, receivedErr)
	}
	done <- struct{}{}
}

// This test checks the proper functioning of the handleCollationBodyRequests goroutine
// by listening to the responseSent channel which occurs after successful
// construction and sending of a response via p2p.
func TestHandleCollationBodyRequests(t *testing.T) {
	h := internal.NewLogHandler(t)
	log.Root().SetHandler(h)

	shardChainDB := database.NewShardKV()
	server, err := p2p.NewServer()
	if err != nil {
		t.Fatalf("Unable to setup p2p server: %v", err)
	}

	body := []byte{1, 2, 3, 4, 5}
	shardID := big.NewInt(0)
	chunkRoot := types.DeriveSha(sharding.Chunks(body))
	period := big.NewInt(0)
	proposerAddress := common.BytesToAddress([]byte{})
	signer := &mockSigner{}

	header := sharding.NewCollationHeader(shardID, &chunkRoot, period, &proposerAddress, nil)
	sig, err := signer.Sign(header.Hash())
	if err != nil {
		t.Fatalf("Could not sign header: %v", err)
	}

	// Adds the signature to the header before calculating the hash used for db lookups.
	header.AddSig(sig)

	// Stores the collation into the inmemory kv store shardChainDB.
	collation := sharding.NewCollation(header, body, nil)

	shard := sharding.NewShard(shardID, shardChainDB)

	if err := shard.SaveCollation(collation); err != nil {
		t.Fatalf("Could not store collation in shardChainDB: %v", err)
	}

	feed := server.Feed(messages.CollationBodyRequest{})
	msgChan := make(chan p2p.Message, 1)
	done := make(chan struct{})
	errChan := make(chan error)
	sub := feed.Subscribe(msgChan)

	go handleCollationBodyRequests(&mockSigner{}, shard, server, msgChan, done, sub, errChan)

	msg := p2p.Message{
		Peer: p2p.Peer{},
		Data: messages.CollationBodyRequest{
			ChunkRoot: &chunkRoot,
			ShardID:   shardID,
			Period:    period,
			Proposer:  &proposerAddress,
		},
	}
	msgChan <- msg
	done <- struct{}{}
	// TODO: get this to work without timeout.
	time.Sleep(time.Second * 5)
	h.VerifyLogMsg(fmt.Sprintf("Received p2p request of type: %T", p2p.Message{}))
	h.VerifyLogMsg(fmt.Sprintf("Responding to p2p request with collation with headerHash: %v", header.Hash().Hex()))
}
