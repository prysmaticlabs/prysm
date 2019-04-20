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
func (v *validator) ProposeBlock(ctx context.Context, slot uint64, idx string) {
	if slot == params.BeaconConfig().GenesisSlot {
		log.Info("Assigned to genesis slot, skipping proposal")
		return
	}
	ctx, span := trace.StartSpan(ctx, "validator.ProposeBlock")
	defer span.End()
	span.AddAttributes(trace.StringAttribute("validator", fmt.Sprintf("%#x", v.keys[idx].PublicKey.Marshal())))
	truncatedPk := idx
	if len(idx) > 12 {
		truncatedPk = idx[:12]
	}
	log.WithFields(logrus.Fields{"validator": truncatedPk}).Info("Performing a beacon block proposal...")
	// 1. Fetch data from Beacon Chain node.
	// Get current head beacon block.
	headBlock, err := v.beaconClient.CanonicalHead(ctx, &ptypes.Empty{})
	if err != nil {
		log.WithError(err).Error("Failed to fetch CanonicalHead")
		return
	}
	parentTreeRoot, err := hashutil.HashBeaconBlock(headBlock)
	if err != nil {
		log.WithError(err).Error("Failed to hash parent block")
		return
	}

	// Get validator ETH1 deposits which have not been included in the beacon chain.
	pDepResp, err := v.beaconClient.PendingDeposits(ctx, &ptypes.Empty{})
	if err != nil {
		log.WithError(err).Error("Failed to get pendings deposits")
		return
	}

	// Get ETH1 data.
	eth1DataResp, err := v.beaconClient.Eth1Data(ctx, &ptypes.Empty{})
	if err != nil {
		log.WithError(err).Error("Failed to get ETH1 data")
		return
	}

	// Retrieve the current fork data from the beacon node.
	fork, err := v.beaconClient.ForkData(ctx, &ptypes.Empty{})
	if err != nil {
		log.WithError(err).Error("Failed to get fork data from beacon node's state")
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
	epochSignature := v.keys[idx].SecretKey.Sign(buf, domain)

	// Fetch pending attestations seen by the beacon node.
	attResp, err := v.proposerClient.PendingAttestations(ctx, &pb.PendingAttestationsRequest{
		FilterReadyForInclusion: true,
		ProposalBlockSlot:       slot,
	})
	if err != nil {
		log.WithError(err).Error("Failed to fetch pending attestations from the beacon node")
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
		log.WithFields(logrus.Fields{
			"block":     proto.MarshalTextString(block),
			"validator": truncatedPk,
		}).WithError(err).Error("Not proposing! Unable to compute state root")
		return
	}
	block.StateRootHash32 = resp.GetStateRoot()

	// 4. Sign the complete block.
	// TODO(1366): BLS sign block
	block.Signature = nil

	// 5. Broadcast to the network via beacon chain node.
	blkResp, err := v.proposerClient.ProposeBlock(ctx, block)
	if err != nil {
		log.WithError(err).WithFields(logrus.Fields{
			"validator": truncatedPk,
		}).Error("Failed to propose block")
		return
	}
	span.AddAttributes(
		trace.StringAttribute("blockRoot", fmt.Sprintf("%#x", blkResp.BlockRootHash32)),
		trace.Int64Attribute("numDeposits", int64(len(block.Body.Deposits))),
		trace.Int64Attribute("numAttestations", int64(len(block.Body.Attestations))),
	)
	log.WithFields(logrus.Fields{
		"slot":            block.Slot - params.BeaconConfig().GenesisSlot,
		"blockRoot":       fmt.Sprintf("%#x", blkResp.BlockRootHash32),
		"numAttestations": len(block.Body.Attestations),
		"numDeposits":     len(block.Body.Deposits),
		"validator":       truncatedPk,
	}).Info("Proposed new beacon block")
}
