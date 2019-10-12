package client

// Validator client proposer functions.
import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/prysmaticlabs/go-ssz"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// ProposeBlock A new beacon block for a given slot. This method collects the
// previous beacon block, any pending deposits, and ETH1 data from the beacon
// chain node to construct the new block. The new block is then processed with
// the state root computation, and finally signed by the validator before being
// sent back to the beacon node for broadcasting.
func (v *validator) ProposeBlock(ctx context.Context, slot uint64, pubKey [48]byte) {
	if slot == 0 {
		log.Info("Assigned to genesis slot, skipping proposal")
		return
	}
	ctx, span := trace.StartSpan(ctx, "validator.ProposeBlock")
	defer span.End()

	span.AddAttributes(trace.StringAttribute("validator", fmt.Sprintf("%#x", pubKey)))
	log := log.WithField("pubKey", fmt.Sprintf("%#x", bytesutil.Trunc(pubKey[:])))

	epoch := slot / params.BeaconConfig().SlotsPerEpoch
	domain, err := v.validatorClient.DomainData(ctx, &pb.DomainRequest{Epoch: epoch, Domain: params.BeaconConfig().DomainRandao})
	if err != nil {
		log.WithError(err).Error("Failed to get domain data from beacon node")
		return
	}
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf, epoch)
	randaoReveal := v.keys[pubKey].SecretKey.Sign(buf, domain.SignatureDomain)

	b, err := v.proposerClient.RequestBlock(ctx, &pb.BlockRequest{
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
	root, err := ssz.SigningRoot(b)
	if err != nil {
		log.WithError(err).Error("Failed to sign block")
		return
	}
	signature := v.keys[pubKey].SecretKey.Sign(root[:], domain.SignatureDomain)
	b.Signature = signature.Marshal()

	// Broadcast network the signed block via beacon chain node.
	blkResp, err := v.proposerClient.ProposeBlock(ctx, b)
	if err != nil {
		log.WithError(err).Error("Failed to propose block")
		return
	}

	span.AddAttributes(
		trace.StringAttribute("blockRoot", fmt.Sprintf("%#x", blkResp.BlockRoot)),
		trace.Int64Attribute("numDeposits", int64(len(b.Body.Deposits))),
		trace.Int64Attribute("numAttestations", int64(len(b.Body.Attestations))),
	)

	blkRoot := fmt.Sprintf("%#x", bytesutil.Trunc(blkResp.BlockRoot))
	log.WithFields(logrus.Fields{
		"slot":            b.Slot,
		"blockRoot":       blkRoot,
		"numAttestations": len(b.Body.Attestations),
		"numDeposits":     len(b.Body.Deposits),
	}).Info("Submitted new block")
}
