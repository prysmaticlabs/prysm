package p2p

import (
	"testing"

	"github.com/ethereum/go-ethereum/sharding"
	internal "github.com/ethereum/go-ethereum/sharding/internal"
)

// Ensure that server implements service.
var _ = sharding.Service(&Server{})

func TestLifecycle(t *testing.T) {
	h := internal.NewLogHandler(t)
	logger.SetHandler(h)

	s, err := NewServer()
	if err != nil {
		t.Fatalf("Could not start a new server: %v", err)
	}

	s.Start()
	h.VerifyLogMsg("Starting shardp2p server")

	s.Stop()
	h.VerifyLogMsg("Stopping shardp2p server")

	// The context should have been cancelled.
	if s.ctx.Err() == nil {
		t.Error("Context was not cancelled")
	}
}
