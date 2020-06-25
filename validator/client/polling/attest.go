package polling

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
	"github.com/prysmaticlabs/prysm/validator/client/metrics"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"go.opencensus.io/trace"
)

// SubmitAttestation completes the validator client's attester responsibility at a given slot.
// It fetches the latest beacon block head along with the latest canonical beacon state
// information in order to sign the block and include information about the validator's
// participation in voting on the block.
func (v *validator) SubmitAttestation(ctx context.Context, slot uint64, pubKey [48]byte) {
	ctx, span := trace.StartSpan(ctx, "validator.SubmitAttestation")
	defer span.End()
	span.AddAttributes(trace.StringAttribute("validator", fmt.Sprintf("%#x", pubKey)))

	fmtKey := fmt.Sprintf("%#x", pubKey[:])
	log := log.WithField("pubKey", fmt.Sprintf("%#x", bytesutil.Trunc(pubKey[:]))).WithField("slot", slot)
	duty, err := v.duty(pubKey)
	if err != nil {
		log.WithError(err).Error("Could not fetch validator assignment")
		if v.emitAccountMetrics {
			metrics.ValidatorAttestFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}
	if len(duty.Committee) == 0 {
		log.Debug("Empty committee for validator duty, not attesting")
		return
	}

	v.waitToSlotOneThird(ctx, slot)

	req := &ethpb.AttestationDataRequest{
		Slot:           slot,
		CommitteeIndex: duty.CommitteeIndex,
	}
	data, err := v.validatorClient.GetAttestationData(ctx, req)
	if err != nil {
		log.WithError(err).Error("Could not request attestation to sign at slot")
		if v.emitAccountMetrics {
			metrics.ValidatorAttestFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}

	indexedAtt := &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{duty.ValidatorIndex},
		Data:             data,
	}
	if err := v.preAttSignValidations(ctx, indexedAtt, pubKey); err != nil {
		log.WithError(err).Error("Failed to attest")
		return
	}

	sig, err := v.signAtt(ctx, pubKey, data)
	if err != nil {
		log.WithError(err).Error("Could not sign attestation")
		if v.emitAccountMetrics {
			metrics.ValidatorAttestFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}

	var indexInCommittee uint64
	var found bool
	for i, vID := range duty.Committee {
		if vID == duty.ValidatorIndex {
			indexInCommittee = uint64(i)
			found = true
			break
		}
	}
	if !found {
		log.Errorf("Validator ID %d not found in committee of %v", duty.ValidatorIndex, duty.Committee)
		if v.emitAccountMetrics {
			metrics.ValidatorAttestFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}

	aggregationBitfield := bitfield.NewBitlist(uint64(len(duty.Committee)))
	aggregationBitfield.SetBitAt(indexInCommittee, true)
	attestation := &ethpb.Attestation{
		Data:            data,
		AggregationBits: aggregationBitfield,
		Signature:       sig,
	}

	attResp, err := v.validatorClient.ProposeAttestation(ctx, attestation)
	if err != nil {
		log.WithError(err).Error("Could not submit attestation to beacon node")
		if v.emitAccountMetrics {
			metrics.ValidatorAttestFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}

	if err := v.saveAttesterIndexToData(data, duty.ValidatorIndex); err != nil {
		log.WithError(err).Error("Could not save validator index for logging")
		if v.emitAccountMetrics {
			metrics.ValidatorAttestFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}

	span.AddAttributes(
		trace.Int64Attribute("slot", int64(slot)),
		trace.StringAttribute("attestationHash", fmt.Sprintf("%#x", attResp.AttestationDataRoot)),
		trace.Int64Attribute("committeeIndex", int64(data.CommitteeIndex)),
		trace.StringAttribute("blockRoot", fmt.Sprintf("%#x", data.BeaconBlockRoot)),
		trace.Int64Attribute("justifiedEpoch", int64(data.Source.Epoch)),
		trace.Int64Attribute("targetEpoch", int64(data.Target.Epoch)),
		trace.StringAttribute("bitfield", fmt.Sprintf("%#x", aggregationBitfield)),
	)

	indexedAtt.Signature = sig
	if err := v.postAttSignUpdate(ctx, indexedAtt, pubKey); err != nil {
		log.WithError(err).Fatal("Failed post attestation signing updates")
		return
	}

	if v.emitAccountMetrics {
		metrics.ValidatorAttestSuccessVec.WithLabelValues(fmtKey).Inc()
	}
}

// Given the validator public key, this gets the validator assignment.
func (v *validator) duty(pubKey [48]byte) (*ethpb.DutiesResponse_Duty, error) {
	if v.duties == nil {
		return nil, errors.New("no duties for validators")
	}

	for _, duty := range v.duties.Duties {
		if bytes.Equal(pubKey[:], duty.PublicKey) {
			return duty, nil
		}
	}

	return nil, fmt.Errorf("pubkey %#x not in duties", bytesutil.Trunc(pubKey[:]))
}

// Given validator's public key, this returns the signature of an attestation data.
func (v *validator) signAtt(ctx context.Context, pubKey [48]byte, data *ethpb.AttestationData) ([]byte, error) {
	domain, err := v.domainData(ctx, data.Target.Epoch, params.BeaconConfig().DomainBeaconAttester[:])
	if err != nil {
		return nil, err
	}

	root, err := helpers.ComputeSigningRoot(data, domain.SignatureDomain)
	if err != nil {
		return nil, err
	}

	var sig *bls.Signature
	if protectingKeymanager, supported := v.keyManager.(keymanager.ProtectingKeyManager); supported {
		sig, err = protectingKeymanager.SignAttestation(pubKey, bytesutil.ToBytes32(domain.SignatureDomain), data)
	} else {
		sig, err = v.keyManager.Sign(pubKey, root)
	}
	if err != nil {
		return nil, err
	}

	return sig.Marshal(), nil
}

// For logging, this saves the last submitted attester index to its attestation data. The purpose of this
// is to enhance attesting logs to be readable when multiple validator keys ran in a single client.
func (v *validator) saveAttesterIndexToData(data *ethpb.AttestationData, index uint64) error {
	v.attLogsLock.Lock()
	defer v.attLogsLock.Unlock()

	h, err := hashutil.HashProto(data)
	if err != nil {
		return err
	}

	if v.attLogs[h] == nil {
		v.attLogs[h] = &attSubmitted{data, []uint64{}, []uint64{}}
	}
	v.attLogs[h] = &attSubmitted{data, append(v.attLogs[h].attesterIndices, index), []uint64{}}

	return nil
}

// waitToSlotOneThird waits until one third through the current slot period
// such that head block for beacon node can get updated.
func (v *validator) waitToSlotOneThird(ctx context.Context, slot uint64) {
	_, span := trace.StartSpan(ctx, "validator.waitToSlotOneThird")
	defer span.End()

	delay := slotutil.DivideSlotBy(3 /* a third of the slot duration */)
	startTime := slotutil.SlotStartTime(v.genesisTime, slot)
	finalTime := startTime.Add(delay)
	time.Sleep(roughtime.Until(finalTime))
}
