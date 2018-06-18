package simulator

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"testing"

	ethereum "github.com/ethereum/go-ethereum"
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

func (f *faultyReader) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	return nil, fmt.Errorf("cannot fetch block by number")
}

func (f *faultyReader) SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error) {
	return nil, nil
}

func TestSimulateNotaryRequests(t *testing.T) {
	h := internal.NewLogHandler(t)
	log.Root().SetHandler(h)

	shardID := 0
	server, err := p2p.NewServer()
	if err != nil {
		t.Fatalf("Unable to setup p2p server: %v", err)
	}

	syncer, err := NewSimulator(params.DefaultConfig, &mainchain.SMCClient{}, server, shardID)
	if err != nil {
		t.Fatalf("Unable to setup simulator service: %v", err)
	}

	feed := server.Feed(messages.CollationBodyRequest{})
	reader := &faultyReader{}

	go syncer.simulateNotaryRequests(&mainchain.SMCClient{}, reader, feed)

	select {
	case err := <-syncer.errChan:
		expectedErr := "Could not fetch current block number"
		if !strings.Contains(err.Error(), expectedErr) {
			t.Errorf("Expected error did not match. want: %v, got: %v", expectedErr, err)
		}
		if err := syncer.Stop(); err != nil {
			t.Fatalf("Unable to stop simulator service: %v", err)
		}
		h.VerifyLogMsg("Stopping simulator service")

		// The context should have been canceled.
		if syncer.ctx.Err() == nil {
			t.Error("Context was not canceled")
		}
	}
}
