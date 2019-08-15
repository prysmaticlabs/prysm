package sync

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared"
)

var _ = shared.Service(&RegularSync{})

// RegularSync service is responsible for handling all run time p2p related operations as the
// main entry point for network messages.
type RegularSync struct {
	ctx context.Context
	p2p p2p.P2P
}

// Start the regular sync service by initializing all of the p2p sync handlers.
func (r *RegularSync) Start() {
	// Register RPC handlers.
	r.registerRPC(
		"/eth2/beacon_chain/req/hello/1",
		&pb.Hello{},
		r.helloRPCHandler,
	)
	r.registerRPC(
		"/eth2/beacon_chain/req/goodbye/1",
		nil,
		notImplementedRPCHandler, // TODO(3147): Implement.
	)
	r.registerRPC(
		"/eth2/beacon_chain/req/recent_beacon_blocks/1",
		nil,
		notImplementedRPCHandler, // TODO(3147): Implement.
	)
	r.registerRPC(
		"/eth2/beacon_chain/req/beacon_blocks/1",
		nil,
		notImplementedRPCHandler, // TODO(3147): Implement.
	)

	// Register PubSub subscribers.
	r.subscribe(
		"/eth2/beacon_block",
		noopValidator,
		notImplementedSubHandler, // TODO(3147): Implement.
	)
	r.subscribe(
		"/eth2/beacon_attestation",
		noopValidator,
		notImplementedSubHandler, // TODO(3147): Implement.
	)
	r.subscribe(
		"/eth2/voluntary_exit",
		noopValidator,
		notImplementedSubHandler, // TODO(3147): Implement.
	)
	r.subscribe(
		"/eth2/proposer_slashing",
		noopValidator,
		notImplementedSubHandler, // TODO(3147): Implement.
	)
	r.subscribe(
		"/eth2/attester_slashing",
		noopValidator,
		notImplementedSubHandler, // TODO(3147): Implement.
	)
}

// Stop the regular sync service.
func (r *RegularSync) Stop() error {
	return nil
}

// Status of the currently running regular sync service.
func (r *RegularSync) Status() error {
	return nil
}
