package client

// Validator client proposer functions.
import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
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

	// Sign randao reveal, it's used to request block from beacon node
	epoch := slot / params.BeaconConfig().SlotsPerEpoch
	randaoReveal, err := v.signRandaoReveal(ctx, pubKey, epoch)
	if err != nil {
		log.WithError(err).Error("Failed to sign randao reveal")
		return
	}

	// Request block from beacon node
	b, err := v.proposerClient.RequestBlock(ctx, &pb.BlockRequest{
		Slot:         slot,
		RandaoReveal: randaoReveal,
		Graffiti:     []byte(v.graffiti),
	})
	if err != nil {
		log.WithError(err).Error("Failed to request block from beacon node")
		return
	}

	history, err := v.db.ProposalHistory(pubKey[:])
	if err != nil {
		log.WithError(err).Error("Failed to get proposal history")
		return
	}

	if HasProposedForEpoch(history, epoch) {
		log.WithField("epoch", epoch).Warn("Tried to sign a double proposal, rejected")
		return
	}

	// Sign returned block from beacon node
	sig, err := v.signBlock(ctx, pubKey, epoch, b)
	if err != nil {
		log.WithError(err).Error("Failed to sign block")
		return
	}
	b.Signature = sig

	history = SetProposedForEpoch(history, epoch)
	if err := v.db.SaveProposalHistory(pubKey[:], history); err != nil {
		log.WithError(err).Error("Failed to save updated proposal history")
		return
	}

	// Propose and broadcast block via beacon node
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

	res, err := v.validatorClient.ValidatorIndex(ctx, &pb.ValidatorIndexRequest{PublicKey: pubKey[:]})
	if err != nil {
		log.WithError(err).Error("Failed to get validator index")
		return
	}

	log.WithField("signature", fmt.Sprintf("%#x", b.Signature)).Debug("block signature")
	blkRoot := fmt.Sprintf("%#x", bytesutil.Trunc(blkResp.BlockRoot))
	log.WithFields(logrus.Fields{
		"slot":            b.Slot,
		"blockRoot":       blkRoot,
		"numAttestations": len(b.Body.Attestations),
		"numDeposits":     len(b.Body.Deposits),
		"proposerIndex":   res.Index,
	}).Info("Submitted new block")
}

// Sign randao reveal with randao domain and private key.
func (v *validator) signRandaoReveal(ctx context.Context, pubKey [48]byte, epoch uint64) ([]byte, error) {
	domain, err := v.validatorClient.DomainData(ctx, &pb.DomainRequest{Epoch: epoch, Domain: params.BeaconConfig().DomainRandao})
	if err != nil {
		return nil, errors.Wrap(err, "could not get domain data")
	}
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf, epoch)
	randaoReveal := v.keys[pubKey].SecretKey.Sign(buf, domain.SignatureDomain)
	return randaoReveal.Marshal(), nil
}

// Sign block with proposer domain and private key.
func (v *validator) signBlock(ctx context.Context, pubKey [48]byte, epoch uint64, b *ethpb.BeaconBlock) ([]byte, error) {
	domain, err := v.validatorClient.DomainData(ctx, &pb.DomainRequest{Epoch: epoch, Domain: params.BeaconConfig().DomainBeaconProposer})
	if err != nil {
		return nil, errors.Wrap(err, "could not get domain data")
	}
	root, err := ssz.SigningRoot(b)
	if err != nil {
		return nil, errors.Wrap(err, "could not get signing root")
	}
	sig := v.keys[pubKey].SecretKey.Sign(root[:], domain.SignatureDomain)
	return sig.Marshal(), nil
}


// HasProposedForEpoch returns whether a validators proposal history has been marked for the entered epoch.
// If the request is more in the future than what the history contains, it will return false.
// If the request is from the past, and likely previously pruned it will return false.
func HasProposedForEpoch(history *slashpb.ProposalHistory, epoch uint64) bool {
	wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod
	// Previously pruned, but to be safe we should return false.
	if int(epoch) <= int(history.LatestEpochWritten)-int(wsPeriod) {
		return false
	}
	// Accessing future proposals that haven't been marked yet. Needs to return false.
	if epoch > history.LatestEpochWritten {
		return false
	}
	return history.EpochBits.BitAt(epoch % wsPeriod)
}

// SetProposedForEpoch updates the proposal history to mark the indicated epoch in the bitlist
// and updates the last epoch written if needed.
// Returns the modified proposal history.
func SetProposedForEpoch(history *slashpb.ProposalHistory, epoch uint64) *slashpb.ProposalHistory {
	wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod

	if epoch > history.LatestEpochWritten {
		// If the history is empty, just update the latest written and mark the epoch.
		// This is for the first run of a validator.
		if history.EpochBits.Count() < 1 {
			history.LatestEpochWritten = epoch
			history.EpochBits.SetBitAt(epoch%wsPeriod, true)
			return history
		}
		// If the epoch to mark is ahead of latest written epoch, override the old votes and mark the requested epoch.
		// Limit the overwriting to one weak subjectivity period as further is not needed.
		maxToWrite := history.LatestEpochWritten + wsPeriod
		for i := history.LatestEpochWritten + 1; i < epoch && i <= maxToWrite; i++ {
			history.EpochBits.SetBitAt(i%wsPeriod, false)
		}
		history.LatestEpochWritten = epoch
	}
	history.EpochBits.SetBitAt(epoch%wsPeriod, true)
	return history
}