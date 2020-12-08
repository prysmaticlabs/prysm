package client

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
	attestinghistory "github.com/prysmaticlabs/prysm/validator/slashing-protection/local/attesting-history"
)

var failedAttLocalProtectionErr = "attempted to make slashable attestation, rejected by local slashing protection"
var failedPreAttSignExternalErr = "attempted to make slashable attestation, rejected by external slasher service"
var failedPostAttSignExternalErr = "external slasher service detected a submitted slashable attestation"

func (v *validator) preAttSignValidations(ctx context.Context, indexedAtt *ethpb.IndexedAttestation, pubKey [48]byte) error {
	fmtKey := fmt.Sprintf("%#x", pubKey[:])
	v.attesterHistoryByPubKeyLock.RLock()
	attesterHistory, ok := v.attesterHistoryByPubKey[pubKey]
	v.attesterHistoryByPubKeyLock.RUnlock()
	if !ok {
		AttestationMapMiss.Inc()
		attesterHistoryMap, err := v.db.AttestationHistoryForPubKeysV2(ctx, [][48]byte{pubKey})
		if err != nil {
			return errors.Wrap(err, "could not get attester history")
		}
		attesterHistory, ok = attesterHistoryMap[pubKey]
		if !ok {
			log.WithField("publicKey", fmtKey).Debug("Could not get local slashing protection data for validator in pre validation")
		}
	} else {
		AttestationMapHit.Inc()
	}
	_, sr, err := v.getDomainAndSigningRoot(ctx, indexedAtt.Data)
	if err != nil {
		log.WithError(err).Error("Could not get domain and signing root from attestation")
		return err
	}
	slashable, err := attestinghistory.IsNewAttSlashable(
		ctx,
		attesterHistory,
		indexedAtt.Data.Source.Epoch,
		indexedAtt.Data.Target.Epoch,
		sr,
	)
	if err != nil {
		return errors.Wrap(err, "could not check if attestation is slashable")
	}
	if slashable {
		if v.emitAccountMetrics {
			ValidatorAttestFailVec.WithLabelValues(fmtKey).Inc()
		}
		return errors.New(failedAttLocalProtectionErr)
	}
	if featureconfig.Get().SlasherProtection && v.protector != nil {
		if !v.protector.CheckAttestationSafety(ctx, indexedAtt) {
			if v.emitAccountMetrics {
				ValidatorAttestFailVecSlasher.WithLabelValues(fmtKey).Inc()
			}
			return errors.New(failedPreAttSignExternalErr)
		}
	}

	// Based on EIP3076, validator should refuse to sign any attestation with source epoch less
	// than the minimum source epoch present in that signer’s attestations.
	lowestSourceEpoch, exists, err := v.db.LowestSignedSourceEpoch(ctx, pubKey)
	if err != nil {
		return err
	}
	if exists && lowestSourceEpoch > indexedAtt.Data.Source.Epoch {
		return fmt.Errorf("could not sign attestation lower than lowest source epoch in db, %d > %d", lowestSourceEpoch, indexedAtt.Data.Source.Epoch)
	}
	// Based on EIP3076, validator should refuse to sign any attestation with target epoch less
	// than or equal to the minimum target epoch present in that signer’s attestations.
	lowestTargetEpoch, exists, err := v.db.LowestSignedTargetEpoch(ctx, pubKey)
	if err != nil {
		return err
	}
	if exists && lowestTargetEpoch >= indexedAtt.Data.Target.Epoch {
		return fmt.Errorf("could not sign attestation lower than lowest target epoch in db, %d >= %d", lowestTargetEpoch, indexedAtt.Data.Target.Epoch)
	}

	return nil
}

func (v *validator) postAttSignUpdate(ctx context.Context, indexedAtt *ethpb.IndexedAttestation, pubKey [48]byte, signingRoot [32]byte) error {
	fmtKey := fmt.Sprintf("%#x", pubKey[:])
	v.attesterHistoryByPubKeyLock.Lock()
	defer v.attesterHistoryByPubKeyLock.Unlock()
	attesterHistory, ok := v.attesterHistoryByPubKey[pubKey]
	if !ok {
		AttestationMapMiss.Inc()
		attesterHistoryMap, err := v.db.AttestationHistoryForPubKeysV2(ctx, [][48]byte{pubKey})
		if err != nil {
			return errors.Wrap(err, "could not get attester history")
		}
		attesterHistory, ok = attesterHistoryMap[pubKey]
		if !ok {
			log.WithField("publicKey", fmtKey).Debug("Could not get local slashing protection data for validator in post validation")
		}
	} else {
		AttestationMapHit.Inc()
	}
	slashable, err := attestinghistory.IsNewAttSlashable(
		ctx,
		attesterHistory,
		indexedAtt.Data.Source.Epoch,
		indexedAtt.Data.Target.Epoch,
		signingRoot,
	)
	if err != nil {
		return errors.Wrap(err, "could not check if attestation is slashable")
	}
	if slashable {
		if v.emitAccountMetrics {
			ValidatorAttestFailVec.WithLabelValues(fmtKey).Inc()
		}
		return errors.New(failedAttLocalProtectionErr)
	}
	newHistory, err := kv.MarkAllAsAttestedSinceLatestWrittenEpoch(
		ctx,
		attesterHistory,
		indexedAtt.Data.Target.Epoch,
		&kv.HistoryData{
			Source:      indexedAtt.Data.Source.Epoch,
			SigningRoot: signingRoot[:],
		},
	)
	if err != nil {
		return errors.Wrapf(err, "could not mark epoch %d as attested", indexedAtt.Data.Target.Epoch)
	}
	v.attesterHistoryByPubKey[pubKey] = newHistory

	if featureconfig.Get().SlasherProtection && v.protector != nil {
		if !v.protector.CommitAttestation(ctx, indexedAtt) {
			if v.emitAccountMetrics {
				ValidatorAttestFailVecSlasher.WithLabelValues(fmtKey).Inc()
			}
			return errors.New(failedPostAttSignExternalErr)
		}
	}

	// Save source and target epochs to satisfy EIP3076 requirements.
	// The DB methods below will replace the lowest epoch in DB if necessary.
	if err := v.db.SaveLowestSignedSourceEpoch(ctx, pubKey, indexedAtt.Data.Source.Epoch); err != nil {
		return err
	}
	return v.db.SaveLowestSignedTargetEpoch(ctx, pubKey, indexedAtt.Data.Target.Epoch)
}
