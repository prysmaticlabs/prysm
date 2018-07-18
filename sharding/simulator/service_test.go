package simulator

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/prysmaticlabs/geth-sharding/sharding/mainchain"
	"github.com/prysmaticlabs/geth-sharding/sharding/p2p"
	"github.com/prysmaticlabs/geth-sharding/sharding/params"
	"github.com/prysmaticlabs/geth-sharding/sharding/types"
	log "github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func init() {
	log.SetLevel(log.DebugLevel)
	log.SetOutput(ioutil.Discard)
}

var _ = types.Service(&Simulator{})

type faultyReader struct{}
type goodReader struct{}

type faultySMCCaller struct{}
type goodSMCCaller struct{}

func (f *faultySMCCaller) CollationRecords(opts *bind.CallOpts, arg0 *big.Int, arg1 *big.Int) (struct {
	ChunkRoot [32]byte
	Proposer  common.Address
	IsElected bool
	Signature [32]byte
}, error) {
	res := new(struct {
		ChunkRoot [32]byte
		Proposer  common.Address
		IsElected bool
		Signature [32]byte
	})
	return *res, errors.New("error fetching collation record")
}

func (g *goodSMCCaller) CollationRecords(opts *bind.CallOpts, arg0 *big.Int, arg1 *big.Int) (struct {
	ChunkRoot [32]byte
	Proposer  common.Address
	IsElected bool
	Signature [32]byte
}, error) {
	res := new(struct {
		ChunkRoot [32]byte
		Proposer  common.Address
		IsElected bool
		Signature [32]byte
	})
	body := []byte{1, 2, 3, 4, 5}
	res.ChunkRoot = [32]byte(gethTypes.DeriveSha(types.Chunks(body)))
	res.Proposer = common.BytesToAddress([]byte{})
	res.IsElected = false
	return *res, nil
}

func (f *faultyReader) BlockByNumber(ctx context.Context, number *big.Int) (*gethTypes.Block, error) {
	return nil, fmt.Errorf("cannot fetch block by number")
}

func (f *faultyReader) SubscribeNewHead(ctx context.Context, ch chan<- *gethTypes.Header) (ethereum.Subscription, error) {
	return nil, nil
}

func (g *goodReader) BlockByNumber(ctx context.Context, number *big.Int) (*gethTypes.Block, error) {
	return gethTypes.NewBlock(&gethTypes.Header{Number: big.NewInt(0)}, nil, nil, nil), nil
}

func (g *goodReader) SubscribeNewHead(ctx context.Context, ch chan<- *gethTypes.Header) (ethereum.Subscription, error) {
	return nil, nil
}

func TestStartStop(t *testing.T) {
	hook := logTest.NewGlobal()

	shardID := 0
	server, err := p2p.NewServer()
	if err != nil {
		t.Fatalf("Unable to setup p2p server: %v", err)
	}

	simulator, err := NewSimulator(params.DefaultConfig, &mainchain.SMCClient{}, server, shardID, 1*time.Second)
	if err != nil {
		t.Fatalf("Unable to setup simulator service: %v", err)
	}

	simulator.Start()
	msg := hook.LastEntry().Message
	if msg != "Starting simulator service" {
		t.Errorf("incorrect log, expected %s, got %s", "Starting simulator service", msg)
	}

	if err := simulator.Stop(); err != nil {
		t.Fatalf("Unable to stop simulator service: %v", err)
	}

	msg = hook.LastEntry().Message
	if msg != "Stopping simulator service" {
		t.Errorf("incorrect log, expected %s, got %s", "Stopping simulator service", msg)
	}

	// The context should have been canceled.
	if simulator.ctx.Err() == nil {
		t.Error("Context was not canceled")
	}
	hook.Reset()
}

// This test uses a faulty chain reader in order to trigger an error
// in the simulateNotaryRequests goroutine when reading the block number from
// the mainchain via RPC.
func TestSimulateNotaryRequests_FaultyReader(t *testing.T) {
	hook := logTest.NewGlobal()

	shardID := 0
	server, err := p2p.NewServer()
	if err != nil {
		t.Fatalf("Unable to setup p2p server: %v", err)
	}

	simulator, err := NewSimulator(params.DefaultConfig, &mainchain.SMCClient{}, server, shardID, 0)
	if err != nil {
		t.Fatalf("Unable to setup simulator service: %v", err)
	}

	delayChan := make(chan time.Time)
	doneChan := make(chan struct{})
	exitRoutine := make(chan bool)
	go func() {
		simulator.simulateNotaryRequests(&goodSMCCaller{}, &faultyReader{}, delayChan, doneChan)
		<-exitRoutine
	}()

	delayChan <- time.Time{}
	doneChan <- struct{}{}

	msg := hook.LastEntry().Message
	want := "Could not fetch current block number: cannot fetch block by number"
	if msg != want {
		t.Errorf("incorrect log, expected %s, got %s", want, msg)
	}

	exitRoutine <- true
	hook.Reset()
}

// This test uses a faulty SMCCaller in order to trigger an error
// in the simulateNotaryRequests goroutine when reading the collation records
// from the SMC.
func TestSimulateNotaryRequests_FaultyCaller(t *testing.T) {
	hook := logTest.NewGlobal()

	shardID := 0
	server, err := p2p.NewServer()
	if err != nil {
		t.Fatalf("Unable to setup p2p server: %v", err)
	}

	simulator, err := NewSimulator(params.DefaultConfig, &mainchain.SMCClient{}, server, shardID, 0)
	if err != nil {
		t.Fatalf("Unable to setup simulator service: %v", err)
	}

	delayChan := make(chan time.Time)
	doneChan := make(chan struct{})
	exitRoutine := make(chan bool)
	go func() {
		simulator.simulateNotaryRequests(&faultySMCCaller{}, &goodReader{}, delayChan, doneChan)
		<-exitRoutine
	}()

	delayChan <- time.Time{}
	doneChan <- struct{}{}

	msg := hook.AllEntries()[0].Message
	want := "Error constructing collation body request: could not fetch collation record from SMC: error fetching collation record"
	if msg != want {
		t.Errorf("incorrect log, expected %s, got %s", want, msg)
	}

	exitRoutine <- true
	hook.Reset()
}

// This test checks the proper functioning of the simulateNotaryRequests goroutine
// by listening to the requestSent channel which occurs after successful
// construction and sending of a request via p2p.
func TestSimulateNotaryRequests(t *testing.T) {
	hook := logTest.NewGlobal()

	shardID := 0
	server, err := p2p.NewServer()
	if err != nil {
		t.Fatalf("Unable to setup p2p server: %v", err)
	}

	simulator, err := NewSimulator(params.DefaultConfig, &mainchain.SMCClient{}, server, shardID, 0)
	if err != nil {
		t.Fatalf("Unable to setup simulator service: %v", err)
	}

	delayChan := make(chan time.Time)
	doneChan := make(chan struct{})
	exitRoutine := make(chan bool)

	go func() {
		simulator.simulateNotaryRequests(&goodSMCCaller{}, &goodReader{}, delayChan, doneChan)
		<-exitRoutine
	}()

	delayChan <- time.Time{}
	doneChan <- struct{}{}

	msg := hook.Entries[1].Message
	want := "Sent request for collation body via a shardp2p broadcast"
	if msg != want {
		t.Errorf("incorrect log, expected %s, got %s", want, msg)
	}

	exitRoutine <- true
	hook.Reset()
}

// This test verifies actor simulator can successfully broadcast
// transactions to rest of the peers.
func TestBroadcastTransactions(t *testing.T) {
	hook := logTest.NewGlobal()

	shardID := 0
	server, err := p2p.NewServer()
	if err != nil {
		t.Fatalf("Unable to setup p2p server: %v", err)
	}

	simulator, err := NewSimulator(params.DefaultConfig, &mainchain.SMCClient{}, server, shardID, 1*time.Second)
	if err != nil {
		t.Fatalf("Unable to setup simulator service: %v", err)
	}

	delayChan := make(chan time.Time)
	doneChan := make(chan struct{})
	exitRoutine := make(chan bool)

	go func() {
		simulator.broadcastTransactions(delayChan, doneChan)
		<-exitRoutine
	}()

	delayChan <- time.Time{}
	doneChan <- struct{}{}

	msg := hook.Entries[1].Message
	want := "Transaction broadcasted"
	if !strings.Contains(msg, want) {
		t.Errorf("incorrect log, expected %s, got %s", want, msg)
	}

	exitRoutine <- true
	hook.Reset()
}
