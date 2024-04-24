package client

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/v5/async"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/v5/config/features"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v5/monitoring/tracing"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	validatorpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1/validator-client"
	prysmTime "github.com/prysmaticlabs/prysm/v5/time"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
	"github.com/prysmaticlabs/prysm/v5/validator/client/iface"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
	"golang.org/x/exp/maps"
)

// GetAttestationData fetches the attestation data that should be signed
// and submitted to the beacon node by an attester.
func (v *validator) GetAttestationData(ctx context.Context, slot primitives.Slot, pubkey [fieldparams.BLSPubkeyLength]byte) *ethpb.AttestationData {
	ctx, span := trace.StartSpan(ctx, "validator.GetAttestationData")
	defer span.End()
	span.AddAttributes(trace.StringAttribute("validator", fmt.Sprintf("%#x", pubkey)))

	v.waitOneThirdOrValidBlock(ctx, slot)

	var b strings.Builder
	if err := b.WriteByte(byte(iface.RoleAttester)); err != nil {
		log.WithError(err).Error("Could not write role byte")
		tracing.AnnotateError(span, err)
		return nil
	}
	_, err := b.Write(pubkey[:])
	if err != nil {
		log.WithError(err).Error("Could not write pubkey bytes")
		tracing.AnnotateError(span, err)
		return nil
	}

	lock := async.NewMultilock(b.String())
	lock.Lock()
	defer lock.Unlock()

	fmtKey := fmt.Sprintf("%#x", pubkey[:])
	log := log.WithField("pubkey", fmt.Sprintf("%#x", bytesutil.Trunc(pubkey[:]))).WithField("slot", slot)
	duty, err := v.duty(pubkey)
	if err != nil {
		log.WithError(err).Error("Could not fetch validator duty")
		if v.emitAccountMetrics {
			ValidatorAttestFailVec.WithLabelValues(fmtKey).Inc()
		}
		tracing.AnnotateError(span, err)
		return nil
	}
	if len(duty.Committee) == 0 {
		log.Debug("Empty committee for validator duty, not attesting")
		return nil
	}

	req := &ethpb.AttestationDataRequest{
		Slot:           slot,
		CommitteeIndex: duty.CommitteeIndex,
	}
	data, err := v.validatorClient.GetAttestationData(ctx, req)
	if err != nil {
		log.WithError(err).Error("Could not get attestation data")
		if v.emitAccountMetrics {
			ValidatorAttestFailVec.WithLabelValues(fmtKey).Inc()
		}
		tracing.AnnotateError(span, err)
		return nil
	}
	return data
}

// SubmitAttestations submits attestations for one slot to the beacon node.
// If there is an issue while preparing an attestation from the provided attestation data,
// then the attestation is ignored.
func (v *validator) SubmitAttestations(
	ctx context.Context,
	slot primitives.Slot,
	pubkeys [][fieldparams.BLSPubkeyLength]byte,
	attData []*ethpb.AttestationData,
) {
	ctx, span := trace.StartSpan(ctx, "validator.SubmitAttestations")
	defer span.End()

	if len(pubkeys) != len(attData) {
		log.Errorf(
			"Skipping submitting attestations. The number of pubkeys %d does not match the number of attestation data %d",
			len(pubkeys),
			len(attData),
		)
		return
	}
	if len(pubkeys) == 0 {
		return
	}

	fmtKeys := make([]string, len(pubkeys))
	for i, pk := range pubkeys {
		fmtKeys[i] = fmt.Sprintf("%#x", pk)
	}
	span.AddAttributes(trace.StringAttribute("pubkeys", strings.Join(fmtKeys, ",")))

	// A map of attestations to be submitted.
	// Because not all attestation data passed into the function might end up being submitted,
	// the key stores the corresponding index of the attestation data slice.
	// This allows to easily find values associated with the submitted att.
	atts := make(map[int]*ethpb.Attestation, len(pubkeys))

	for i := 0; i < len(pubkeys); i++ {
		sig, _, err := v.signAtt(ctx, pubkeys[i], attData[i], slot)
		if err != nil {
			log.WithError(err).Error("Could not sign attestation")
			if v.emitAccountMetrics {
				ValidatorAttestFailVec.WithLabelValues(fmtKeys[i]).Inc()
			}
			tracing.AnnotateError(span, err)
			continue
		}

		duty, err := v.duty(pubkeys[i])
		if err != nil {
			log.WithError(err).Error("Could not get duty")
			if v.emitAccountMetrics {
				ValidatorAttestFailVec.WithLabelValues(fmtKeys[i]).Inc()
			}
			tracing.AnnotateError(span, err)
			continue
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
				ValidatorAttestFailVec.WithLabelValues(fmtKeys[i]).Inc()
			}
			continue
		}

		aggregationBitfield := bitfield.NewBitlist(uint64(len(duty.Committee)))
		aggregationBitfield.SetBitAt(indexInCommittee, true)
		atts[i] = &ethpb.Attestation{
			Data:            attData[i],
			AggregationBits: aggregationBitfield,
			Signature:       sig,
		}

		indexedAtt := &ethpb.IndexedAttestation{
			AttestingIndices: []uint64{uint64(duty.ValidatorIndex)},
			Data:             attData[i],
			Signature:        sig,
		}
		_, signingRoot, err := v.getDomainAndSigningRoot(ctx, indexedAtt.Data)
		if err != nil {
			log.WithError(err).Error("Could not get domain and signing root from attestation")
			if v.emitAccountMetrics {
				ValidatorAttestFailVec.WithLabelValues(fmtKeys[i]).Inc()
			}
			tracing.AnnotateError(span, err)
			continue
		}
		if err = v.db.SlashableAttestationCheck(ctx, indexedAtt, pubkeys[i], signingRoot, v.emitAccountMetrics, ValidatorAttestFailVec); err != nil {
			log.WithError(err).Error("Failed attestation slashing protection check")
			log.WithFields(
				attestationLogFields(pubkeys[i], indexedAtt),
			).Debug("Failed attestation slashing protection details")
			tracing.AnnotateError(span, err)
			continue
		}
	}

	attResp, err := v.validatorClient.SubmitAttestations(ctx, maps.Values(atts))
	if err != nil {
		log.WithError(err).Error("Could not submit attestation to beacon node")
		if v.emitAccountMetrics {
			for _, pk := range fmtKeys {
				ValidatorAttestFailVec.WithLabelValues(pk).Inc()
			}
		}
		tracing.AnnotateError(span, err)
		return
	}

	rootIndex := 0
	for i, a := range atts {
		if err = v.saveSubmittedAtt(attData[i], pubkeys[i][:], false); err != nil {
			log.WithError(err).Error("Could not save validator index for logging")
			tracing.AnnotateError(span, err)
			continue
		}

		span.AddAttributes(
			trace.Int64Attribute("slot", int64(slot)), // lint:ignore uintcast -- This conversion is OK for tracing.
			trace.StringAttribute("attestationDataRoot", fmt.Sprintf("%#x", attResp[rootIndex].AttestationDataRoot)),
			trace.Int64Attribute("committeeIndex", int64(attData[i].CommitteeIndex)),
			trace.StringAttribute("blockRoot", fmt.Sprintf("%#x", attData[i].BeaconBlockRoot)),
			trace.Int64Attribute("sourceEpoch", int64(attData[i].Source.Epoch)),
			trace.Int64Attribute("targetEpoch", int64(attData[i].Target.Epoch)),
			trace.StringAttribute("bitfield", fmt.Sprintf("%#x", a.AggregationBits)),
		)

		if v.emitAccountMetrics {
			ValidatorAttestSuccessVec.WithLabelValues(fmtKeys[i]).Inc()
			ValidatorAttestedSlotsGaugeVec.WithLabelValues(fmtKeys[i]).Set(float64(slot))
		}

		rootIndex++
	}
}

// Given the validator public key, this gets the validator assignment.
func (v *validator) duty(pubKey [fieldparams.BLSPubkeyLength]byte) (*ethpb.DutiesResponse_Duty, error) {
	v.dutiesLock.RLock()
	defer v.dutiesLock.RUnlock()
	if v.duties == nil {
		return nil, errors.New("no duties for validators")
	}

	for _, duty := range v.duties.CurrentEpochDuties {
		if bytes.Equal(pubKey[:], duty.PublicKey) {
			return duty, nil
		}
	}

	return nil, fmt.Errorf("pubkey %#x not in duties", bytesutil.Trunc(pubKey[:]))
}

// Given validator's public key, this function returns the signature of an attestation data and its signing root.
func (v *validator) signAtt(ctx context.Context, pubKey [fieldparams.BLSPubkeyLength]byte, data *ethpb.AttestationData, slot primitives.Slot) ([]byte, [32]byte, error) {
	domain, root, err := v.getDomainAndSigningRoot(ctx, data)
	if err != nil {
		return nil, [32]byte{}, err
	}
	sig, err := v.keyManager.Sign(ctx, &validatorpb.SignRequest{
		PublicKey:       pubKey[:],
		SigningRoot:     root[:],
		SignatureDomain: domain.SignatureDomain,
		Object:          &validatorpb.SignRequest_AttestationData{AttestationData: data},
		SigningSlot:     slot,
	})
	if err != nil {
		return nil, [32]byte{}, err
	}

	return sig.Marshal(), root, nil
}

func (v *validator) getDomainAndSigningRoot(ctx context.Context, data *ethpb.AttestationData) (*ethpb.DomainResponse, [32]byte, error) {
	domain, err := v.domainData(ctx, data.Target.Epoch, params.BeaconConfig().DomainBeaconAttester[:])
	if err != nil {
		return nil, [32]byte{}, err
	}
	root, err := signing.ComputeSigningRoot(data, domain.SignatureDomain)
	if err != nil {
		return nil, [32]byte{}, err
	}
	return domain, root, nil
}

// highestSlot returns the highest slot with a valid block seen by the validator
func (v *validator) highestSlot() primitives.Slot {
	v.highestValidSlotLock.Lock()
	defer v.highestValidSlotLock.Unlock()
	return v.highestValidSlot
}

// setHighestSlot sets the highest slot with a valid block seen by the validator
func (v *validator) setHighestSlot(slot primitives.Slot) {
	v.highestValidSlotLock.Lock()
	defer v.highestValidSlotLock.Unlock()
	if slot > v.highestValidSlot {
		v.highestValidSlot = slot
		v.slotFeed.Send(slot)
	}
}

// waitOneThirdOrValidBlock waits until (a) or (b) whichever comes first:
//
//	(a) the validator has received a valid block that is the same slot as input slot
//	(b) one-third of the slot has transpired (SECONDS_PER_SLOT / 3 seconds after the start of slot)
func (v *validator) waitOneThirdOrValidBlock(ctx context.Context, slot primitives.Slot) {
	ctx, span := trace.StartSpan(ctx, "validator.waitOneThirdOrValidBlock")
	defer span.End()

	// Don't need to wait if requested slot is the same as highest valid slot.
	if slot <= v.highestSlot() {
		return
	}

	delay := slots.DivideSlotBy(3 /* a third of the slot duration */)
	startTime := slots.StartTime(v.genesisTime, slot)
	finalTime := startTime.Add(delay)
	wait := prysmTime.Until(finalTime)
	if wait <= 0 {
		return
	}
	t := time.NewTimer(wait)
	defer t.Stop()

	ch := make(chan primitives.Slot, 1)
	sub := v.slotFeed.Subscribe(ch)
	defer sub.Unsubscribe()

	for {
		select {
		case s := <-ch:
			if features.Get().AttestTimely {
				if slot <= s {
					return
				}
			}
		case <-ctx.Done():
			tracing.AnnotateError(span, ctx.Err())
			return
		case <-sub.Err():
			log.Error("Subscriber closed, exiting goroutine")
			return
		case <-t.C:
			return
		}
	}
}

func attestationLogFields(pubKey [fieldparams.BLSPubkeyLength]byte, indexedAtt *ethpb.IndexedAttestation) logrus.Fields {
	return logrus.Fields{
		"pubkey":         fmt.Sprintf("%#x", pubKey),
		"slot":           indexedAtt.Data.Slot,
		"committeeIndex": indexedAtt.Data.CommitteeIndex,
		"blockRoot":      fmt.Sprintf("%#x", indexedAtt.Data.BeaconBlockRoot),
		"sourceEpoch":    indexedAtt.Data.Source.Epoch,
		"sourceRoot":     fmt.Sprintf("%#x", indexedAtt.Data.Source.Root),
		"targetEpoch":    indexedAtt.Data.Target.Epoch,
		"targetRoot":     fmt.Sprintf("%#x", indexedAtt.Data.Target.Root),
		"signature":      fmt.Sprintf("%#x", indexedAtt.Signature),
	}
}
