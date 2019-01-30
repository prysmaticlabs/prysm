package client

// Validator client proposer functions.

import (
	"context"

	"github.com/opentracing/opentracing-go"

	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

// ProposeBlock
//
// WIP - not done.
func (v *validator) ProposeBlock(ctx context.Context, slot uint64) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "validator.ProposeBlock")
	defer span.Finish()

	// 1. Get current head beacon block.
	headBlock := v.blockThing.HeadBlock()
	parentHash, err := hashutil.HashBeaconBlock(headBlock)
	if err != nil {
		log.Errorf("Failed to hash parent block: %v", err)
		return
	}

	// 2. Construct block
	block := &pbp2p.BeaconBlock{
		Slot:               slot,
		ParentRootHash32:   parentHash[:], // tree root? in ssz pkg
		RandaoRevealHash32: nil,           // TODO: generate randao reveal
		Eth1Data: &pbp2p.Eth1Data{ // TODO(raul): will write rpc
			DepositRootHash32: nil,
			BlockHash32:       nil,
		},
		Body: &pbp2p.BeaconBlockBody{
			Attestations: v.attestationPool.PendingAttestations(),
			// TODO: slashings
			ProposerSlashings: nil, // TODO later
			AttesterSlashings: nil, // TODO later
			Deposits:          nil, // TODO(raul): fetch from gRPC
			Exits:             nil, // TODO: later
		},
	}

	resp, err := v.proposerClient.ComputeStateRoot(ctx, block)
	if err != nil {
		log.Errorf("Unable to compute state root: %v", err)
	}

	block.StateRootHash32 = resp.GetStateRoot()

	// TODO: sign block
	block.Signature = nil

	v.p2p.Broadcast(block)
}
