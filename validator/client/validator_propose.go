package client

// Validator client proposer functions.
import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var (
	validatorProposeSuccessVec = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "validator",
			Name:      "successful_proposals",
		},
		[]string{
			// validator pubkey
			"pkey",
		},
	)
	validatorProposeFailVec = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "validator",
			Name:      "failed_proposals",
		},
		[]string{
			// validator pubkey
			"pkey",
		},
	)
)

// ProposeBlock A new beacon block for a given slot. This method collects the
// previous beacon block, any pending deposits, and ETH1 data from the beacon
// chain node to construct the new block. The new block is then processed with
// the state root computation, and finally signed by the validator before being
// sent back to the beacon node for broadcasting.
func (v *validator) ProposeBlock(ctx context.Context, slot uint64, pubKey [48]byte) {
	if slot == 0 {
		log.Debug("Assigned to genesis slot, skipping proposal")
		return
	}
	ctx, span := trace.StartSpan(ctx, "validator.ProposeBlock")
	defer span.End()
	fmtKey := fmt.Sprintf("%#x", pubKey[:8])

	span.AddAttributes(trace.StringAttribute("validator", fmt.Sprintf("%#x", pubKey)))
	log := log.WithField("pubKey", fmt.Sprintf("%#x", bytesutil.Trunc(pubKey[:])))

	// Sign randao reveal, it's used to request block from beacon node
	epoch := slot / params.BeaconConfig().SlotsPerEpoch
	randaoReveal, err := v.signRandaoReveal(ctx, pubKey, epoch)
	if err != nil {
		log.WithError(err).Error("Failed to sign randao reveal")
		if v.emitAccountMetrics {
			validatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}

	// Request block from beacon node
	b, err := v.validatorClient.GetBlock(ctx, &ethpb.BlockRequest{
		Slot:         slot,
		RandaoReveal: randaoReveal,
		Graffiti:     v.graffiti,
	})
	if err != nil {
		log.WithError(err).Error("Failed to request block from beacon node")
		if v.emitAccountMetrics {
			validatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}

	if featureconfig.Get().ProtectProposer {
		history, err := v.db.ProposalHistory(ctx, pubKey[:])
		if err != nil {
			log.WithError(err).Error("Failed to get proposal history")
			if v.emitAccountMetrics {
				validatorProposeFailVec.WithLabelValues(fmtKey).Inc()
			}
			return
		}

		if HasProposedForEpoch(history, epoch) {
			log.WithField("epoch", epoch).Warn("Tried to sign a double proposal, rejected")
			if v.emitAccountMetrics {
				validatorProposeFailVec.WithLabelValues(fmtKey).Inc()
			}
			return
		}
	}

	// Sign returned block from beacon node
	sig, err := v.signBlock(ctx, pubKey, epoch, b)
	if err != nil {
		log.WithError(err).Error("Failed to sign block")
		if v.emitAccountMetrics {
			validatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}
	blk := &ethpb.SignedBeaconBlock{
		Block:     b,
		Signature: sig,
	}

	// Propose and broadcast block via beacon node
	blkResp, err := v.validatorClient.ProposeBlock(ctx, blk)
	if err != nil {
		log.WithError(err).Error("Failed to propose block")
		if v.emitAccountMetrics {
			validatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}

	if featureconfig.Get().ProtectProposer {
		history, err := v.db.ProposalHistory(ctx, pubKey[:])
		if err != nil {
			log.WithError(err).Error("Failed to get proposal history")
			if v.emitAccountMetrics {
				validatorProposeFailVec.WithLabelValues(fmtKey).Inc()
			}
			return
		}
		history = SetProposedForEpoch(history, epoch)
		if err := v.db.SaveProposalHistory(ctx, pubKey[:], history); err != nil {
			log.WithError(err).Error("Failed to save updated proposal history")
			if v.emitAccountMetrics {
				validatorProposeFailVec.WithLabelValues(fmtKey).Inc()
			}
			return
		}
	}

	if v.emitAccountMetrics {
		validatorProposeSuccessVec.WithLabelValues(fmtKey).Inc()
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

// ProposeExit --
func (v *validator) ProposeExit(ctx context.Context, exit *ethpb.VoluntaryExit) error {
	return errors.New("unimplemented")
}

// Sign randao reveal with randao domain and private key.
func (v *validator) signRandaoReveal(ctx context.Context, pubKey [48]byte, epoch uint64) ([]byte, error) {
	domain, err := v.domainData(ctx, epoch, params.BeaconConfig().DomainRandao)

	if err != nil {
		return nil, errors.Wrap(err, "could not get domain data")
	}
	var buf [32]byte
	binary.LittleEndian.PutUint64(buf[:], epoch)
	randaoReveal, err := v.keyManager.Sign(pubKey, buf, domain.SignatureDomain)
	if err != nil {
		return nil, errors.Wrap(err, "could not sign reveal")
	}
	return randaoReveal.Marshal(), nil
}

// Sign block with proposer domain and private key.
func (v *validator) signBlock(ctx context.Context, pubKey [48]byte, epoch uint64, b *ethpb.BeaconBlock) ([]byte, error) {
	domain, err := v.domainData(ctx, epoch, params.BeaconConfig().DomainBeaconProposer)
	if err != nil {
		return nil, errors.Wrap(err, "could not get domain data")
	}
	root, err := ssz.HashTreeRoot(b)
	if err != nil {
		return nil, errors.Wrap(err, "could not get signing root")
	}
	sig, err := v.keyManager.Sign(pubKey, root, domain.SignatureDomain)
	if err != nil {
		return nil, errors.Wrap(err, "could not get signing root")
	}
	return sig.Marshal(), nil
}

// HasProposedForEpoch returns whether a validators proposal history has been marked for the entered epoch.
// If the request is more in the future than what the history contains, it will return false.
// If the request is from the past, and likely previously pruned it will return false.
func HasProposedForEpoch(history *slashpb.ProposalHistory, epoch uint64) bool {
	wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod
	// Previously pruned, we should return false.
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
