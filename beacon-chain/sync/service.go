package sync

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared"
)

var _ = shared.Service(&RegularSync{})

type RegularSync struct {
	ctx context.Context
	p2p p2p.Composite
}

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
		nil,
		noopValidator,
		notImplementedSubHandler, // TODO(3147): Implement.
	)
	r.subscribe(
		"/eth2/beacon_attestation",
		nil,
		noopValidator,
		notImplementedSubHandler, // TODO(3147): Implement.
	)
	r.subscribe(
		"/eth2/voluntary_exit",
		nil,
		noopValidator,
		notImplementedSubHandler, // TODO(3147): Implement.
	)
	r.subscribe(
		"/eth2/proposer_slashing",
		nil,
		noopValidator,
		notImplementedSubHandler, // TODO(3147): Implement.
	)
	r.subscribe(
		"/eth2/attester_slashing",
		nil,
		noopValidator,
		notImplementedSubHandler, // TODO(3147): Implement.
	)
}

func (r *RegularSync) Stop() error {
	return nil
}

func (r *RegularSync) Status() error {
	return nil
}
