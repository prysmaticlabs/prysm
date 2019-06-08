package client

// Validator client proposer functions.
import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// ProposeBlock A new beacon block for a given slot. This method collects the
// previous beacon block, any pending deposits, and ETH1 data from the beacon
// chain node to construct the new block. The new block is then processed with
// the state root computation, and finally signed by the validator before being
// sent back to the beacon node for broadcasting.
func (v *validator) ProposeBlock(ctx context.Context, slot uint64, pk string) {
	if slot == 0 {
		log.Info("Assigned to genesis slot, skipping proposal")
		return
	}
	ctx, span := trace.StartSpan(ctx, "validator.ProposeBlock")
	defer span.End()
	span.AddAttributes(trace.StringAttribute("validator", fmt.Sprintf("%#x", v.keys[pk].PublicKey.Marshal())))
	truncatedPk := bytesutil.Trunc([]byte(pk))

	log.WithFields(logrus.Fields{"validator": truncatedPk}).Info("Performing a beacon block proposal...")

	// Generate a randao reveal by signing the block's slot with validator's private key.
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

	// Retrieve the current fork data from the beacon node.
	domain, err := v.validatorClient.DomainData(ctx, &pb.DomainRequest{Epoch: epoch})
	if err != nil {
		log.WithError(err).Error("Failed to get domain data from beacon node's state")
		return
	}
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf, epoch)

	randaoReveal := v.keys[pk].SecretKey.Sign(buf, domain.SignatureDomain)

	b, err := v.proposerClient.RequestBlock(ctx, &pb.BlockRequest{
		Slot:         slot,
		RandaoReveal: randaoReveal.Marshal(),
	})
	if err != nil {
		log.WithError(err).Error("Failed to request block from beacon node")
		return
	}

	// Sign the requested block.
	// TODO(1366): BLS sign block
	b.Signature = nil

	// Broadcast network the signed block via beacon chain node.
	blkResp, err := v.proposerClient.ProposeBlock(ctx, b)
	if err != nil {
		log.WithError(err).WithFields(logrus.Fields{
			"validator": truncatedPk,
		}).Error("Failed to propose block")
		return
	}

	span.AddAttributes(
		trace.StringAttribute("blockRoot", fmt.Sprintf("%#x", blkResp.BlockRoot)),
		trace.Int64Attribute("numDeposits", int64(len(b.Body.Deposits))),
		trace.Int64Attribute("numAttestations", int64(len(b.Body.Attestations))),
	)

	log.WithFields(logrus.Fields{
		"validator":       truncatedPk,
		"slot":            b.Slot,
		"blockRoot":       fmt.Sprintf("%#x", blkResp.BlockRoot),
		"numAttestations": len(b.Body.Attestations),
		"numDeposits":     len(b.Body.Deposits),
	}).Info("Proposed new beacon block")
}
