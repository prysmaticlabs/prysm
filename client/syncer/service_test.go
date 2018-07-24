package syncer

import (
	"fmt"
	"io/ioutil"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/prysmaticlabs/prysm/client/database"
	"github.com/prysmaticlabs/prysm/client/mainchain"
	"github.com/prysmaticlabs/prysm/client/params"
	"github.com/prysmaticlabs/prysm/client/types"
	pb "github.com/prysmaticlabs/prysm/proto/sharding/v1"
	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

var _ = shared.Service(&Syncer{})

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
}

func TestStop(t *testing.T) {
	hook := logTest.NewGlobal()

	config := &database.ShardDBConfig{Name: "", DataDir: "", InMemory: true}
	shardChainDB, err := database.NewShardDB(config)
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

	feed := server.Feed(pb.CollationBodyRequest{})
	syncer.msgChan = make(chan p2p.Message)
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
	hook.Reset()
}

// This test checks the proper functioning of the handleCollationBodyRequests goroutine
// by listening to the responseSent channel which occurs after successful
// construction and sending of a response via p2p.
func TestHandleCollationBodyRequests(t *testing.T) {
	hook := logTest.NewGlobal()

	config := &database.ShardDBConfig{Name: "", DataDir: "", InMemory: true}
	shardChainDB, err := database.NewShardDB(config)
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

	feed := server.Feed(pb.CollationBodyRequest{})

	syncer.msgChan = make(chan p2p.Message)
	syncer.bodyRequests = feed.Subscribe(syncer.msgChan)

	doneChan := make(chan struct{})
	exitRoutine := make(chan bool)

	go func() {
		syncer.HandleCollationBodyRequests(shard, doneChan)
		<-exitRoutine
	}()

	msg := p2p.Message{
		Peer: p2p.Peer{},
		Data: &pb.CollationBodyRequest{
			ChunkRoot:       chunkRoot.Bytes(),
			ShardId:         shardID.Uint64(),
			Period:          period.Uint64(),
			ProposerAddress: proposerAddress.Bytes(),
		},
	}
	syncer.msgChan <- msg
	doneChan <- struct{}{}
	exitRoutine <- true

	logMsg := hook.Entries[0].Message
	want := fmt.Sprintf("Received p2p request of type: %T", &pb.CollationBodyRequest{})
	if logMsg != want {
		t.Errorf("incorrect log, expected %s, got %s", want, logMsg)
	}

	logMsg = hook.Entries[3].Message
	want = fmt.Sprintf("Responding to p2p collation request")
	if logMsg != want {
		t.Errorf("incorrect log, expected %s, got %s", want, logMsg)
	}
	hook.Reset()
}
