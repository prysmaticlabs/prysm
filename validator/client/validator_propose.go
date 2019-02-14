package client

// Validator client proposer functions.

import (
	"context"
	"fmt"

	"github.com/opentracing/opentracing-go"

	ptypes "github.com/gogo/protobuf/types"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/ssz"
)

// ProposeBlock A new beacon block for a given slot. This method collects the
// previous beacon block, any pending deposits, and ETH1 data from the beacon
// chain node to construct the new block. The new block is then processed with
// the state root computation, and finally signed by the validator before being
// sent back to the beacon node for broadcasting.
func (v *validator) ProposeBlock(ctx context.Context, slot uint64) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "validator.ProposeBlock")
	defer span.Finish()

	// 1. Fetch data from Beacon Chain node.
	// Get current head beacon block.
	headBlock, err := v.beaconClient.CanonicalHead(ctx, &ptypes.Empty{})
	if err != nil {
		log.Errorf("Failed to fetch CanonicalHead: %v", err)
		return
	}
	parentTreeHash, err := ssz.TreeHash(headBlock)
	if err != nil {
		log.Errorf("Failed to hash parent block: %v", err)
		return
	}

	// Get validator ETH1 deposits which have not been included in the beacon
	// chain.
	pDepResp, err := v.beaconClient.PendingDeposits(ctx, &ptypes.Empty{})
	if err != nil {
		log.Errorf("Failed to get pending pendings: %v", err)
		return
	}

	// Get ETH1 data.
	eth1DataResp, err := v.beaconClient.Eth1Data(ctx, &ptypes.Empty{})
	if err != nil {
		log.Errorf("Failed to get ETH1 data: %v", err)
		return
	}

	// 2. Construct block.
	block := &pbp2p.BeaconBlock{
		Slot:               slot,
		ParentRootHash32:   parentTreeHash[:],
		RandaoRevealHash32: nil, // TODO(1366): generate randao reveal from BLS
		Eth1Data:           eth1DataResp.Eth1Data,
		Body: &pbp2p.BeaconBlockBody{
			Attestations:      v.attestationPool.PendingAttestations(),
			ProposerSlashings: nil, // TODO(1438): Add after operations pool
			AttesterSlashings: nil, // TODO(1438): Add after operations pool
			Deposits:          pDepResp.PendingDeposits,
			Exits:             nil, // TODO(1323): Add validator exits
		},
	}

	// 3. Compute state root transition from parent block to the new block.
	resp, err := v.proposerClient.ComputeStateRoot(ctx, block)
	if err != nil {
		log.Errorf("Unable to compute state root: %v", err)
	}
	block.StateRootHash32 = resp.GetStateRoot()

	// 4. Sign the complete block.
	// TODO(1366): BLS sign block
	block.Signature = nil

	// 5. Broadcast to the network via beacon chain node.
	blkResp, err := v.proposerClient.ProposeBlock(ctx, block)
	if err != nil {
		log.WithField("error", err).Error("Failed to propose block")
		return
	}
	log.WithField("hash", fmt.Sprintf("%#x", blkResp.BlockHash)).Info("Proposed new beacon block")
}
