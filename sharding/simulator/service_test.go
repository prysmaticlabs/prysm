package simulator

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/sharding/p2p/messages"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding"
	internal "github.com/ethereum/go-ethereum/sharding/internal"
	"github.com/ethereum/go-ethereum/sharding/mainchain"
	"github.com/ethereum/go-ethereum/sharding/p2p"
	"github.com/ethereum/go-ethereum/sharding/params"
)

var _ = sharding.Service(&Simulator{})

type faultyReader struct{}
type goodReader struct{}

type faultySMCCaller struct{}
type goodSMCCaller struct{}

func (f *faultySMCCaller) CollationRecords(opts *bind.CallOpts, arg0 *big.Int, arg1 *big.Int) (struct {
	ChunkRoot [32]byte
	Proposer  common.Address
	IsElected bool
}, error) {
	res := new(struct {
		ChunkRoot [32]byte
		Proposer  common.Address
		IsElected bool
	})
	return *res, errors.New("error fetching collation record")
}

func (g *goodSMCCaller) CollationRecords(opts *bind.CallOpts, arg0 *big.Int, arg1 *big.Int) (struct {
	ChunkRoot [32]byte
	Proposer  common.Address
	IsElected bool
}, error) {
	res := new(struct {
		ChunkRoot [32]byte
		Proposer  common.Address
		IsElected bool
	})
	body := []byte{1, 2, 3, 4, 5}
	res.ChunkRoot = [32]byte(types.DeriveSha(sharding.Chunks(body)))
	res.Proposer = common.BytesToAddress([]byte{})
	res.IsElected = false
	return *res, nil
}

func (f *faultyReader) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	return nil, fmt.Errorf("cannot fetch block by number")
}

func (f *faultyReader) SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error) {
	return nil, nil
}

func (g *goodReader) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	return types.NewBlock(&types.Header{Number: big.NewInt(0)}, nil, nil, nil), nil
}

func (g *goodReader) SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error) {
	return nil, nil
}

func TestStartStop(t *testing.T) {
	h := internal.NewLogHandler(t)
	log.Root().SetHandler(h)

	shardID := 0
	server, err := p2p.NewServer()
	if err != nil {
		t.Fatalf("Unable to setup p2p server: %v", err)
	}

	simulator, err := NewSimulator(params.DefaultConfig, &mainchain.SMCClient{}, server, shardID, 0)
	if err != nil {
		t.Fatalf("Unable to setup simulator service: %v", err)
	}

	simulator.Start()

	h.VerifyLogMsg("Starting simulator service")

	if err := simulator.Stop(); err != nil {
		t.Fatalf("Unable to stop simulator service: %v", err)
	}

	h.VerifyLogMsg("Stopping simulator service")

	// The context should have been canceled.
	if simulator.ctx.Err() == nil {
		t.Error("Context was not canceled")
	}
}

// This test verifies actor simulator can successfully broadcast
// transactions to rest of the peers.
func TestBroadcastTransactions(t *testing.T) {
	h := internal.NewLogHandler(t)
	log.Root().SetHandler(h)

	shardID := 0
	server, err := p2p.NewServer()
	if err != nil {
		t.Fatalf("Unable to setup p2p server: %v", err)
	}

	simulator, err := NewSimulator(params.DefaultConfig, &mainchain.SMCClient{}, server, shardID, 5)
	if err != nil {
		t.Fatalf("Unable to setup simulator service: %v", err)
	}

	go simulator.broadcastTransactions()
	// Wait 6 seconds. BroadcastTransactions sends a transaction with random bytes over a 5 second interval.
	time.Sleep(6*time.Second)
	if !strings.Contains(h.Pop().Msg, "Sent transaction") {
		t.Errorf("Failed to broadcast transaction, incorrect log: %v", h.Pop().Msg)
	}

	if err := simulator.Stop(); err != nil {
		t.Fatalf("Unable to stop simulator service: %v", err)
	}
	h.VerifyLogMsg("Stopping simulator service")

	// The context should have been canceled.
	if simulator.ctx.Err() == nil {
		t.Error("Context was not canceled")
	}
}

// This test uses a faulty chain reader in order to trigger an error
// in the simulateNotaryRequests goroutine when reading the block number from
// the mainchain via RPC.
func TestSimulateNotaryRequests_FaultyReader(t *testing.T) {
	h := internal.NewLogHandler(t)
	log.Root().SetHandler(h)

	shardID := 0
	server, err := p2p.NewServer()
	if err != nil {
		t.Fatalf("Unable to setup p2p server: %v", err)
	}

	simulator, err := NewSimulator(params.DefaultConfig, &mainchain.SMCClient{}, server, shardID, 0)
	if err != nil {
		t.Fatalf("Unable to setup simulator service: %v", err)
	}

	feed := server.Feed(messages.CollationBodyRequest{})

	faultyReader := &faultyReader{}
	go simulator.simulateNotaryRequests(&goodSMCCaller{}, faultyReader, feed)

	receivedErr := <-simulator.errChan
	expectedErr := "could not fetch current block number"
	if !strings.Contains(receivedErr.Error(), expectedErr) {
		t.Errorf("Expected error did not match. want: %v, got: %v", expectedErr, receivedErr)
	}
	if err := simulator.Stop(); err != nil {
		t.Fatalf("Unable to stop simulator service: %v", err)
	}
	h.VerifyLogMsg("Stopping simulator service")

	// The context should have been canceled.
	if simulator.ctx.Err() == nil {
		t.Error("Context was not canceled")
	}
}

// This test uses a faulty SMCCaller in order to trigger an error
// in the simulateNotaryRequests goroutine when reading the collation records
// from the SMC.
func TestSimulateNotaryRequests_FaultyCaller(t *testing.T) {
	h := internal.NewLogHandler(t)
	log.Root().SetHandler(h)

	shardID := 0
	server, err := p2p.NewServer()
	if err != nil {
		t.Fatalf("Unable to setup p2p server: %v", err)
	}

	simulator, err := NewSimulator(params.DefaultConfig, &mainchain.SMCClient{}, server, shardID, 0)
	if err != nil {
		t.Fatalf("Unable to setup simulator service: %v", err)
	}

	feed := server.Feed(messages.CollationBodyRequest{})
	reader := &goodReader{}

	go simulator.simulateNotaryRequests(&faultySMCCaller{}, reader, feed)

	receivedErr := <-simulator.errChan
	expectedErr := "error constructing collation body request"
	if !strings.Contains(receivedErr.Error(), expectedErr) {
		t.Errorf("Expected error did not match. want: %v, got: %v", expectedErr, receivedErr)
	}
	if err := simulator.Stop(); err != nil {
		t.Fatalf("Unable to stop simulator service: %v", err)
	}
	h.VerifyLogMsg("Stopping simulator service")

	// The context should have been canceled.
	if simulator.ctx.Err() == nil {
		t.Error("Context was not canceled")
	}
}

// This test checks the proper functioning of the simulateNotaryRequests goroutine
// by listening to the requestSent channel which occurs after successful
// construction and sending of a request via p2p.
func TestSimulateNotaryRequests(t *testing.T) {
	h := internal.NewLogHandler(t)
	log.Root().SetHandler(h)

	shardID := 0
	server, err := p2p.NewServer()
	if err != nil {
		t.Fatalf("Unable to setup p2p server: %v", err)
	}

	simulator, err := NewSimulator(params.DefaultConfig, &mainchain.SMCClient{}, server, shardID, 0)
	if err != nil {
		t.Fatalf("Unable to setup simulator service: %v", err)
	}

	feed := server.Feed(messages.CollationBodyRequest{})
	reader := &goodReader{}

	go simulator.simulateNotaryRequests(&goodSMCCaller{}, reader, feed)

	<-simulator.requestSent
	h.VerifyLogMsg("Sent request for collation body via a shardp2p feed")
	simulator.cancel()
	// The context should have been canceled.
	if simulator.ctx.Err() == nil {
		t.Fatal("Context was not canceled")
	}
}

// TODO: Move this to the utils package along with the handleServiceErrors
// function.
func TestHandleServiceErrors(t *testing.T) {

	h := internal.NewLogHandler(t)
	log.Root().SetHandler(h)

	shardID := 0
	server, err := p2p.NewServer()
	if err != nil {
		t.Fatalf("Unable to setup p2p server: %v", err)
	}

	simulator, err := NewSimulator(params.DefaultConfig, &mainchain.SMCClient{}, server, shardID, 0)
	if err != nil {
		t.Fatalf("Unable to setup simulator service: %v", err)
	}

	go simulator.handleServiceErrors()

	expectedErr := "testing the error channel"
	complete := make(chan int)

	go func() {
		for {
			select {
			case <-simulator.ctx.Done():
				return
			default:
				simulator.errChan <- errors.New(expectedErr)
				complete <- 1
			}
		}
	}()

	<-complete
	simulator.cancel()

	// The context should have been canceled.
	if simulator.ctx.Err() == nil {
		t.Fatal("Context was not canceled")
	}
	time.Sleep(time.Millisecond * 500)
	h.VerifyLogMsg(fmt.Sprintf("Simulator service error: %v", expectedErr))
}
