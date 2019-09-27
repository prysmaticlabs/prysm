package client

// Validator client proposer functions.
import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/prysmaticlabs/go-ssz"
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

	tpk := hex.EncodeToString(v.keys[pk].PublicKey.Marshal())
	span.AddAttributes(trace.StringAttribute("validator", tpk))
	log := log.WithField("pubKey", tpk[:12])

	epoch := slot / params.BeaconConfig().SlotsPerEpoch
	domain, err := v.validatorClient.DomainData(ctx, &pb.DomainRequest{Epoch: epoch, Domain: params.BeaconConfig().DomainRandao})
	if err != nil {
		log.WithError(err).Error("Failed to get domain data from beacon node")
		return
	}
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf, epoch)
	randaoReveal := v.keys[pk].SecretKey.Sign(buf, domain.SignatureDomain)

	block, err := v.proposerClient.RequestBlock(ctx, &pb.BlockRequest{
		Slot:         slot,
		RandaoReveal: randaoReveal.Marshal(),
	})
	if err != nil {
		log.WithError(err).Error("Failed to request block from beacon node")
		return
	}

	domain, err = v.validatorClient.DomainData(ctx, &pb.DomainRequest{Epoch: epoch, Domain: params.BeaconConfig().DomainBeaconProposer})
	if err != nil {
		log.WithError(err).Error("Failed to get domain data from beacon node")
		return
	}
	root, err := ssz.SigningRoot(block)
	if err != nil {
		log.WithError(err).Error("Failed to sign block")
		return
	}
	signature := v.keys[pk].SecretKey.Sign(root[:], domain.SignatureDomain)
	block.Signature = signature.Marshal()

	// Broadcast network the signed block via beacon chain node.
	blkResp, err := v.proposerClient.ProposeBlock(ctx, block)
	if err != nil {
		log.WithError(err).Error("Failed to propose block")
		return
	}

	span.AddAttributes(
		trace.StringAttribute("blockRoot", fmt.Sprintf("%#x", blkResp.BlockRoot)),
		trace.Int64Attribute("numDeposits", int64(len(block.Body.Deposits))),
		trace.Int64Attribute("numAttestations", int64(len(block.Body.Attestations))),
	)

	log.WithFields(logrus.Fields{
		"slot":            block.Slot,
		"blockRoot":       fmt.Sprintf("%#x", blkResp.BlockRoot),
		"numAttestations": len(block.Body.Attestations),
		"numDeposits":     len(block.Body.Deposits),
	}).Info("Proposed new beacon block")
}
