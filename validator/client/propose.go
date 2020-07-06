package client

// Validator client proposer functions.
import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	validatorpb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	km "github.com/prysmaticlabs/prysm/validator/keymanager/v1"
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
		log.Debug("Assigned to genesis slot, skipping proposal")
		return
	}
	ctx, span := trace.StartSpan(ctx, "validator.ProposeBlock")
	defer span.End()
	fmtKey := fmt.Sprintf("%#x", pubKey[:])

	span.AddAttributes(trace.StringAttribute("validator", fmt.Sprintf("%#x", pubKey)))
	log := log.WithField("pubKey", fmt.Sprintf("%#x", bytesutil.Trunc(pubKey[:])))

	// Sign randao reveal, it's used to request block from beacon node
	epoch := slot / params.BeaconConfig().SlotsPerEpoch
	randaoReveal, err := v.signRandaoReveal(ctx, pubKey, epoch)
	if err != nil {
		log.WithError(err).Error("Failed to sign randao reveal")
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
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
		log.WithField("blockSlot", slot).WithError(err).Error("Failed to request block from beacon node")
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}

	if err := v.preBlockSignValidations(ctx, pubKey, b); err != nil {
		log.WithField("slot", b.Slot).WithError(err).Error("Failed block safety check")
		return
	}

	// Sign returned block from beacon node
	sig, err := v.signBlock(ctx, pubKey, epoch, b)
	if err != nil {
		log.WithError(err).Error("Failed to sign block")
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}
	blk := &ethpb.SignedBeaconBlock{
		Block:     b,
		Signature: sig,
	}

	if err := v.postBlockSignUpdate(ctx, pubKey, blk); err != nil {
		log.WithField("slot", blk.Block.Slot).WithError(err).Error("Failed post block signing validations")
		return
	}

	// Propose and broadcast block via beacon node
	blkResp, err := v.validatorClient.ProposeBlock(ctx, blk)
	if err != nil {
		log.WithError(err).Error("Failed to propose block")
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
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

	if v.emitAccountMetrics {
		ValidatorProposeSuccessVec.WithLabelValues(fmtKey).Inc()
	}
}

// ProposeExit --
func (v *validator) ProposeExit(ctx context.Context, exit *ethpb.VoluntaryExit) error {
	return errors.New("unimplemented")
}

// Sign randao reveal with randao domain and private key.
func (v *validator) signRandaoReveal(ctx context.Context, pubKey [48]byte, epoch uint64) ([]byte, error) {
	domain, err := v.domainData(ctx, epoch, params.BeaconConfig().DomainRandao[:])
	if err != nil {
		return nil, errors.Wrap(err, "could not get domain data")
	}

	randaoReveal, err := v.signObject(ctx, pubKey, epoch, domain.SignatureDomain)
	if err != nil {
		return nil, errors.Wrap(err, "could not sign reveal")
	}
	return randaoReveal.Marshal(), nil
}

// Sign block with proposer domain and private key.
func (v *validator) signBlock(ctx context.Context, pubKey [48]byte, epoch uint64, b *ethpb.BeaconBlock) ([]byte, error) {
	domain, err := v.domainData(ctx, epoch, params.BeaconConfig().DomainBeaconProposer[:])
	if err != nil {
		return nil, errors.Wrap(err, "could not get domain data")
	}
	var sig bls.Signature

	if featureconfig.Get().EnableAccountsV2 {
		blockRoot, err := helpers.ComputeSigningRoot(b, domain.SignatureDomain)
		if err != nil {
			return nil, errors.Wrap(err, "could not get signing root")
		}
		sig, err = v.keyManagerV2.Sign(ctx, &validatorpb.SignRequest{
			PublicKey: pubKey[:],
			Data:      blockRoot[:],
		})
		if err != nil {
			return nil, errors.Wrap(err, "could not sign block proposal")
		}
		return sig.Marshal(), nil
	}
	if protectingKeymanager, supported := v.keyManager.(km.ProtectingKeyManager); supported {
		bodyRoot, err := stateutil.BlockBodyRoot(b.Body)
		if err != nil {
			return nil, errors.Wrap(err, "could not get signing root")
		}
		blockHeader := &ethpb.BeaconBlockHeader{
			Slot:          b.Slot,
			ProposerIndex: b.ProposerIndex,
			StateRoot:     b.StateRoot,
			ParentRoot:    b.ParentRoot,
			BodyRoot:      bodyRoot[:],
		}
		sig, err = protectingKeymanager.SignProposal(pubKey, bytesutil.ToBytes32(domain.SignatureDomain), blockHeader)
		if err != nil {
			return nil, errors.Wrap(err, "could not sign block proposal")
		}
	} else {
		blockRoot, err := helpers.ComputeSigningRoot(b, domain.SignatureDomain)
		if err != nil {
			return nil, errors.Wrap(err, "could not get signing root")
		}
		sig, err = v.keyManager.Sign(pubKey, blockRoot)
		if err != nil {
			return nil, errors.Wrap(err, "could not sign block proposal")
		}
	}
	return sig.Marshal(), nil
}
