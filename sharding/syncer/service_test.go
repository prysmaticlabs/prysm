package syncer

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/database"
	"github.com/ethereum/go-ethereum/sharding/p2p"
	"github.com/ethereum/go-ethereum/sharding/params"

	internal "github.com/ethereum/go-ethereum/sharding/internal"
	pb "github.com/ethereum/go-ethereum/sharding/p2p/proto"
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

	syncer, err := NewSyncer(params.DefaultConfig, &mockSigner{}, server, shardChainDB, shardID)
	if err != nil {
		t.Fatalf("Unable to setup sync service: %v", err)
	}

	syncer.Start()

	h.VerifyLogMsg("Starting sync service")

	if err := syncer.Stop(); err != nil {
		t.Fatalf("Unable to stop sync service: %v", err)
	}

	h.VerifyLogMsg("Stopping sync service")

	// The context should have been canceled.
	if syncer.ctx.Err() == nil {
		t.Error("Context was not canceled")
	}
}

// This test uses a faulty Signer interface in order to trigger an error
// in the simulateNotaryRequests goroutine when attempting to sign
// a collation header within the goroutine's internals.
func TestHandleCollationBodyRequests_FaultySigner(t *testing.T) {
	h := internal.NewLogHandler(t)
	log.Root().SetHandler(h)

	shardChainDB := database.NewShardKV()
	shardID := 0
	server, err := p2p.NewServer()
	if err != nil {
		t.Fatalf("Unable to setup p2p server: %v", err)
	}

	syncer, err := NewSyncer(params.DefaultConfig, &mockSigner{}, server, shardChainDB, shardID)
	if err != nil {
		t.Fatalf("Unable to setup syncer service: %v", err)
	}

	feed := server.Feed(pb.CollationBodyRequest{})

	go syncer.handleCollationBodyRequests(&faultySigner{}, feed)

	go func() {
		for {
			select {
			case <-syncer.ctx.Done():
				return
			default:
				msg := p2p.Message{
					Peer: p2p.Peer{},
					Data: pb.CollationBodyRequest{},
				}
				feed.Send(msg)
			}
		}
	}()

	receivedErr := <-syncer.errChan
	expectedErr := "could not construct response"
	if !strings.Contains(receivedErr.Error(), expectedErr) {
		t.Errorf("Expected error did not match. want: %v, got: %v", expectedErr, receivedErr)
	}
	syncer.cancel()
	// The context should have been canceled.
	if syncer.ctx.Err() == nil {
		t.Fatal("Context was not canceled")
	}
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

	syncer, err := NewSyncer(params.DefaultConfig, &mockSigner{}, server, shardChainDB, 0)
	if err != nil {
		t.Fatalf("Unable to setup syncer service: %v", err)
	}

	feed := server.Feed(pb.CollationBodyRequest{})

	go syncer.handleCollationBodyRequests(&mockSigner{}, feed)

	go func() {
		for {
			select {
			case <-syncer.ctx.Done():
				return
			default:
				msg := p2p.Message{
					Peer: p2p.Peer{},
					Data: pb.CollationBodyRequest{
						ChunkRoot:       chunkRoot.Bytes(),
						ShardId:         shardID.Uint64(),
						Period:          period.Uint64(),
						ProposerAddress: proposerAddress.Bytes(),
					},
				}
				feed.Send(msg)
			}
		}
	}()

	<-syncer.responseSent
	h.VerifyLogMsg(fmt.Sprintf("Received p2p request of type: %T", p2p.Message{}))
	h.VerifyLogMsg(fmt.Sprintf("Responding to p2p request with collation with headerHash: %v", header.Hash().Hex()))
	syncer.cancel()
	// The context should have been canceled.
	if syncer.ctx.Err() == nil {
		t.Fatal("Context was not canceled")
	}
}

// TODO: Move this to the utils package along with the handleServiceErrors
// function.
func TestHandleServiceErrors(t *testing.T) {

	h := internal.NewLogHandler(t)
	log.Root().SetHandler(h)

	shardChainDB := database.NewShardKV()
	shardID := 0
	server, err := p2p.NewServer()
	if err != nil {
		t.Fatalf("Unable to setup p2p server: %v", err)
	}

	syncer, err := NewSyncer(params.DefaultConfig, &mockSigner{}, server, shardChainDB, shardID)
	if err != nil {
		t.Fatalf("Unable to setup syncer service: %v", err)
	}

	go syncer.handleServiceErrors()

	expectedErr := "testing the error channel"
	complete := make(chan int)

	go func() {
		for {
			select {
			case <-syncer.ctx.Done():
				return
			default:
				syncer.errChan <- errors.New(expectedErr)
				complete <- 1
			}
		}
	}()

	<-complete
	syncer.cancel()

	// The context should have been canceled.
	if syncer.ctx.Err() == nil {
		t.Fatal("Context was not canceled")
	}
	time.Sleep(time.Millisecond * 500)
	h.VerifyLogMsg(fmt.Sprintf("Sync service error: %v", expectedErr))
}
