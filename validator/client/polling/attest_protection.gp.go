package polling

import (
	"context"
	"fmt"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/validator/client/metrics"
)

func (v *validator) preAttSignValidations(ctx context.Context, indexedAtt *ethpb.IndexedAttestation, pubKey [48]byte) error {
	fmtKey := fmt.Sprintf("%#x", pubKey[:])
	if featureconfig.Get().ProtectAttester {
		v.attesterHistoryByPubKeyLock.RLock()
		attesterHistory := v.attesterHistoryByPubKey[pubKey]
		v.attesterHistoryByPubKeyLock.RUnlock()
		if isNewAttSlashable(attesterHistory, indexedAtt.Data.Source.Epoch, indexedAtt.Data.Target.Epoch) {
			if v.emitAccountMetrics {
				metrics.ValidatorAttestFailVec.WithLabelValues(fmtKey).Inc()
			}
			return fmt.Errorf(
				"attempted to make slashable attestation, rejected by local slasher protection: sourceEpoch=%d targetEpoch=%d",
				indexedAtt.Data.Source.Epoch,
				indexedAtt.Data.Target.Epoch,
			)
		}
	}

	if featureconfig.Get().SlasherProtection && v.protector != nil {
		if !v.protector.CheckAttestationSafety(ctx, indexedAtt) {
			if v.emitAccountMetrics {
				metrics.ValidatorAttestFailVecSlasher.WithLabelValues(fmtKey).Inc()
			}
			return fmt.Errorf(
				"attempted to make slashable attestation, rejected by external slasher service: sourceEpoch=%d targetEpoch=%d",
				indexedAtt.Data.Source.Epoch,
				indexedAtt.Data.Target.Epoch,
			)
		}
	}
	return nil
}

func (v *validator) postAttSignUpdate(ctx context.Context, indexedAtt *ethpb.IndexedAttestation, pubKey [48]byte) error {
	fmtKey := fmt.Sprintf("%#x", pubKey[:])
	if featureconfig.Get().ProtectAttester {
		v.attesterHistoryByPubKeyLock.Lock()
		attesterHistory := v.attesterHistoryByPubKey[pubKey]
		attesterHistory = markAttestationForTargetEpoch(attesterHistory, indexedAtt.Data.Source.Epoch, indexedAtt.Data.Target.Epoch)
		v.attesterHistoryByPubKey[pubKey] = attesterHistory
		v.attesterHistoryByPubKeyLock.Unlock()
	}

	if featureconfig.Get().SlasherProtection && v.protector != nil {
		if !v.protector.CommitAttestation(ctx, indexedAtt) {
			if v.emitAccountMetrics {
				metrics.ValidatorAttestFailVecSlasher.WithLabelValues(fmtKey).Inc()
			}
			return fmt.Errorf("made a slashable attestation, sourceEpoch: %dtargetEpoch: %d  "+
				" found by external slasher service", indexedAtt.Data.Source.Epoch, indexedAtt.Data.Target.Epoch)
		}
	}
	return nil
}
