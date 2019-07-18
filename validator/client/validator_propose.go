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

	epoch := slot / params.BeaconConfig().SlotsPerEpoch
	tpk := hex.EncodeToString(v.keys[pk].PublicKey.Marshal())[:12]

	domain, err := v.validatorClient.DomainData(ctx, &pb.DomainRequest{Epoch: epoch, Domain: params.BeaconConfig().DomainRandao})
	if err != nil {
		log.WithError(err).Error("Failed to get domain data from beacon node")
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
	span.AddAttributes(trace.StringAttribute("validator", tpk))

	domain, err = v.validatorClient.DomainData(ctx, &pb.DomainRequest{Epoch: epoch, Domain: params.BeaconConfig().DomainBeaconProposer})
	if err != nil {
		log.WithError(err).Error("Failed to get domain data from beacon node")
		return
	}
	root, err := ssz.SigningRoot(b)
	if err != nil {
		log.WithError(err).WithFields(logrus.Fields{
			"pubKey": tpk,
		}).Error("Failed to sign block")
		return
	}
	signature := v.keys[pk].SecretKey.Sign(root[:], domain.SignatureDomain)
	b.Signature = signature.Marshal()

	// Broadcast network the signed block via beacon chain node.
	blkResp, err := v.proposerClient.ProposeBlock(ctx, b)
	if err != nil {
		log.WithError(err).WithFields(logrus.Fields{
			"pubKey": tpk,
		}).Error("Failed to propose block")
		return
	}

	span.AddAttributes(
		trace.StringAttribute("blockRoot", fmt.Sprintf("%#x", blkResp.BlockRoot)),
		trace.Int64Attribute("numDeposits", int64(len(b.Body.Deposits))),
		trace.Int64Attribute("numAttestations", int64(len(b.Body.Attestations))),
	)

	log.WithFields(logrus.Fields{
		"pubKey":          tpk,
		"slot":            b.Slot,
		"blockRoot":       fmt.Sprintf("%#x", blkResp.BlockRoot),
		"numAttestations": len(b.Body.Attestations),
		"numDeposits":     len(b.Body.Deposits),
	}).Info("Proposed new beacon block")
}
