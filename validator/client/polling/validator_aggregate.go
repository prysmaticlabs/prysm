package polling

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
	"github.com/prysmaticlabs/prysm/validator/client/metrics"
	"go.opencensus.io/trace"
)

// SubmitAggregateAndProof submits the validator's signed slot signature to the beacon node
// via gRPC. Beacon node will verify the slot signature and determine if the validator is also
// an aggregator. If yes, then beacon node will broadcast aggregated signature and
// proof on the validator's behalf.
func (v *validator) SubmitAggregateAndProof(ctx context.Context, slot uint64, pubKey [48]byte) {
	ctx, span := trace.StartSpan(ctx, "validator.SubmitAggregateAndProof")
	defer span.End()

	span.AddAttributes(trace.StringAttribute("validator", fmt.Sprintf("%#x", pubKey)))
	fmtKey := fmt.Sprintf("%#x", pubKey[:])

	duty, err := v.duty(pubKey)
	if err != nil {
		log.Errorf("Could not fetch validator assignment: %v", err)
		if v.emitAccountMetrics {
			metrics.ValidatorAggFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}

	// Avoid sending beacon node duplicated aggregation requests.
	k := validatorSubscribeKey(slot, duty.CommitteeIndex)
	v.aggregatedSlotCommitteeIDCacheLock.Lock()
	defer v.aggregatedSlotCommitteeIDCacheLock.Unlock()
	if v.aggregatedSlotCommitteeIDCache.Contains(k) {
		return
	}
	v.aggregatedSlotCommitteeIDCache.Add(k, true)

	slotSig, err := v.signSlot(ctx, pubKey, slot)
	if err != nil {
		log.Errorf("Could not sign slot: %v", err)
		if v.emitAccountMetrics {
			metrics.ValidatorAggFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}

	// As specified in spec, an aggregator should wait until two thirds of the way through slot
	// to broadcast the best aggregate to the global aggregate channel.
	// https://github.com/ethereum/eth2.0-specs/blob/v0.9.3/specs/validator/0_beacon-chain-validator.md#broadcast-aggregate
	v.waitToSlotTwoThirds(ctx, slot)

	res, err := v.validatorClient.SubmitAggregateSelectionProof(ctx, &ethpb.AggregateSelectionRequest{
		Slot:           slot,
		CommitteeIndex: duty.CommitteeIndex,
		PublicKey:      pubKey[:],
		SlotSignature:  slotSig,
	})
	if err != nil {
		log.WithField("slot", slot).Errorf("Could not submit slot signature to beacon node: %v", err)
		if v.emitAccountMetrics {
			metrics.ValidatorAggFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}

	sig, err := v.aggregateAndProofSig(ctx, pubKey, res.AggregateAndProof)
	if err != nil {
		log.Errorf("Could not sign aggregate and proof: %v", err)
	}
	_, err = v.validatorClient.SubmitSignedAggregateSelectionProof(ctx, &ethpb.SignedAggregateSubmitRequest{
		SignedAggregateAndProof: &ethpb.SignedAggregateAttestationAndProof{
			Message:   res.AggregateAndProof,
			Signature: sig,
		},
	})
	if err != nil {
		log.Errorf("Could not submit signed aggregate and proof to beacon node: %v", err)
		if v.emitAccountMetrics {
			metrics.ValidatorAggFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}

	if err := v.addIndicesToLog(duty); err != nil {
		log.Errorf("Could not add aggregator indices to logs: %v", err)
		if v.emitAccountMetrics {
			metrics.ValidatorAggFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}
	if v.emitAccountMetrics {
		metrics.ValidatorAggSuccessVec.WithLabelValues(fmtKey).Inc()
	}

}

// This implements selection logic outlined in:
// https://github.com/ethereum/eth2.0-specs/blob/v0.9.3/specs/validator/0_beacon-chain-validator.md#aggregation-selection
func (v *validator) signSlot(ctx context.Context, pubKey [48]byte, slot uint64) ([]byte, error) {
	domain, err := v.domainData(ctx, helpers.SlotToEpoch(slot), params.BeaconConfig().DomainSelectionProof[:])
	if err != nil {
		return nil, err
	}

	sig, err := v.signObject(pubKey, slot, domain.SignatureDomain)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to sign slot")
	}

	return sig.Marshal(), nil
}

// waitToSlotTwoThirds waits until two third through the current slot period
// such that any attestations from this slot have time to reach the beacon node
// before creating the aggregated attestation.
func (v *validator) waitToSlotTwoThirds(ctx context.Context, slot uint64) {
	_, span := trace.StartSpan(ctx, "validator.waitToSlotTwoThirds")
	defer span.End()

	oneThird := slotutil.DivideSlotBy(3 /* one third of slot duration */)
	twoThird := oneThird + oneThird
	delay := twoThird

	startTime := slotutil.SlotStartTime(v.genesisTime, slot)
	finalTime := startTime.Add(delay)
	time.Sleep(roughtime.Until(finalTime))
}

// This returns the signature of validator signing over aggregate and
// proof object.
func (v *validator) aggregateAndProofSig(ctx context.Context, pubKey [48]byte, agg *ethpb.AggregateAttestationAndProof) ([]byte, error) {
	d, err := v.domainData(ctx, helpers.SlotToEpoch(agg.Aggregate.Data.Slot), params.BeaconConfig().DomainAggregateAndProof[:])
	if err != nil {
		return nil, err
	}
	sig, err := v.signObject(pubKey, agg, d.SignatureDomain)
	if err != nil {
		return nil, err
	}

	return sig.Marshal(), nil
}

func (v *validator) addIndicesToLog(duty *ethpb.DutiesResponse_Duty) error {
	v.attLogsLock.Lock()
	defer v.attLogsLock.Unlock()

	for _, log := range v.attLogs {
		if duty.CommitteeIndex == log.data.CommitteeIndex {
			log.aggregatorIndices = append(log.aggregatorIndices, duty.ValidatorIndex)
		}
	}

	return nil
}
