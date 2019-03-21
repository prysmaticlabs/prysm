package client

// Validator client proposer functions.

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/gogo/protobuf/proto"
	ptypes "github.com/gogo/protobuf/types"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/forkutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// ProposeBlock A new beacon block for a given slot. This method collects the
// previous beacon block, any pending deposits, and ETH1 data from the beacon
// chain node to construct the new block. The new block is then processed with
// the state root computation, and finally signed by the validator before being
// sent back to the beacon node for broadcasting.
func (v *validator) ProposeBlock(ctx context.Context, slot uint64) {
	if slot == params.BeaconConfig().GenesisSlot {
		log.Info("Assigned to genesis slot, skipping proposal")
		return
	}
	ctx, span := trace.StartSpan(ctx, "validator.ProposeBlock")
	defer span.End()
	log.Info("Performing a beacon block proposal...")
	// 1. Fetch data from Beacon Chain node.
	// Get current head beacon block.
	headBlock, err := v.beaconClient.CanonicalHead(ctx, &ptypes.Empty{})
	if err != nil {
		log.Errorf("Failed to fetch CanonicalHead: %v", err)
		return
	}
	parentTreeRoot, err := hashutil.HashBeaconBlock(headBlock)
	if err != nil {
		log.Errorf("Failed to hash parent block: %v", err)
		return
	}

	// Get validator ETH1 deposits which have not been included in the beacon chain.
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

	// Retrieve the current fork data from the beacon node.
	fork, err := v.beaconClient.ForkData(ctx, &ptypes.Empty{})
	if err != nil {
		log.Errorf("Failed to get fork data from beacon node's state: %v", err)
		return
	}
	// Then, we generate a RandaoReveal by signing the block's slot information using
	// the validator's private key.
	// epoch_signature = bls_sign(
	//   privkey=validator.privkey,
	//   message_hash=int_to_bytes32(slot_to_epoch(block.slot)),
	//   domain=get_domain(
	//     fork=fork,  # `fork` is the fork object at the slot `block.slot`
	//     epoch=slot_to_epoch(block.slot),
	//	   domain_type=DOMAIN_RANDAO,
	//   )
	// )
	epoch := slot / params.BeaconConfig().SlotsPerEpoch
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf, epoch)
	domain := forkutil.DomainVersion(fork, epoch, params.BeaconConfig().DomainRandao)
	epochSignature := v.key.SecretKey.Sign(buf, domain)

	// Fetch pending attestations seen by the beacon node.
	attResp, err := v.proposerClient.PendingAttestations(ctx, &pb.PendingAttestationsRequest{
		FilterReadyForInclusion: true,
		ProposalBlockSlot:       slot,
	})
	if err != nil {
		log.Errorf("Failed to fetch pending attestations from the beacon node: %v", err)
		return
	}

	// 2. Construct block.
	block := &pbp2p.BeaconBlock{
		Slot:             slot,
		ParentRootHash32: parentTreeRoot[:],
		RandaoReveal:     epochSignature.Marshal(),
		Eth1Data:         eth1DataResp.Eth1Data,
		Body: &pbp2p.BeaconBlockBody{
			Attestations:      attResp.PendingAttestations,
			ProposerSlashings: nil, // TODO(1438): Add after operations pool
			AttesterSlashings: nil, // TODO(1438): Add after operations pool
			Deposits:          pDepResp.PendingDeposits,
			VoluntaryExits:    nil, // TODO(1323): Add validator exits
		},
	}

	// 3. Compute state root transition from parent block to the new block.
	resp, err := v.proposerClient.ComputeStateRoot(ctx, block)
	if err != nil {
		log.WithField(
			"block", proto.MarshalTextString(block),
		).Errorf("Not proposing! Unable to compute state root: %v", err)
		return
	}
	block.StateRootHash32 = resp.GetStateRoot()

	// 4. Sign the complete block.
	// TODO(1366): BLS sign block
	block.Signature = nil

	// 5. Broadcast to the network via beacon chain node.
	blkResp, err := v.proposerClient.ProposeBlock(ctx, block)
	if err != nil {
		log.WithError(err).Error("Failed to propose block")
		return
	}
	log.WithFields(logrus.Fields{
		"blockRoot": fmt.Sprintf("%#x", blkResp.BlockRootHash32),
	}).Info("Proposed new beacon block")
	log.WithFields(logrus.Fields{
		"numAttestations": len(block.Body.Attestations),
		"numDeposits":     len(block.Body.Deposits),
	}).Info("Items included in block")
}
