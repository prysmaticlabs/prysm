package client

import (
	"context"
	"fmt"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
	"go.opencensus.io/trace"
)

// SubmitAggregateAndProof submits the validator's signed slot signature to the beacon node
// via gRPC. Beacon node will verify the slot signature and determine if the validator is also
// an aggregator. If yes, then beacon node will broadcast aggregated signature and
// proof on the validator's behave.
func (v *validator) SubmitAggregateAndProof(ctx context.Context, slot uint64, pubKey [48]byte) {
	ctx, span := trace.StartSpan(ctx, "validator.SubmitAggregateAndProof")
	defer span.End()

	span.AddAttributes(trace.StringAttribute("validator", fmt.Sprintf("%#x", pubKey)))

	duty, err := v.duty(pubKey)
	if err != nil {
		log.Errorf("Could not fetch validator assignment: %v", err)
		return
	}

	slotSig, err := v.signSlot(ctx, pubKey, slot)
	if err != nil {
		log.Errorf("Could not sign slot: %v", err)
		return
	}

	// As specified in spec, an aggregator should wait until two thirds of the way through slot
	// to broadcast the best aggregate to the global aggregate channel.
	// https://github.com/ethereum/eth2.0-specs/blob/v0.9.0/specs/validator/0_beacon-chain-validator.md#broadcast-aggregate
	v.waitToSlotTwoThirds(ctx, slot)

	_, err = v.aggregatorClient.SubmitAggregateAndProof(ctx, &pb.AggregationRequest{
		Slot:           slot,
		CommitteeIndex: duty.CommitteeIndex,
		PublicKey:      pubKey[:],
		SlotSignature:  slotSig,
	})
	if err != nil {
		log.Errorf("Could not submit slot signature to beacon node: %v", err)
		return
	}

	if err := v.addIndicesToLog(ctx, duty.CommitteeIndex, pubKey[:]); err != nil {
		log.Errorf("Could not add aggregator indices to logs: %v", err)
		return
	}
}

// This implements selection logic outlined in:
// https://github.com/ethereum/eth2.0-specs/blob/v0.9.0/specs/validator/0_beacon-chain-validator.md#aggregation-selection
func (v *validator) signSlot(ctx context.Context, pubKey [48]byte, slot uint64) ([]byte, error) {
	domain, err := v.validatorClient.DomainData(ctx, &ethpb.DomainRequest{
		Epoch:  helpers.SlotToEpoch(slot),
		Domain: params.BeaconConfig().DomainBeaconAttester,
	})
	if err != nil {
		return nil, err
	}

	slotRoot, err := ssz.HashTreeRoot(slot)
	if err != nil {
		return nil, err
	}

	sig, err := v.keyManager.Sign(pubKey, slotRoot, domain.SignatureDomain)
	if err != nil {
		return nil, err
	}

	return sig.Marshal(), nil
}

// waitToSlotTwoThirds waits until two third through the current slot period
// such that any attestations from this slot have time to reach the beacon node
// before creating the aggregated attestation.
func (v *validator) waitToSlotTwoThirds(ctx context.Context, slot uint64) {
	_, span := trace.StartSpan(ctx, "validator.waitToSlotTwoThirds")
	defer span.End()

	twoThird := params.BeaconConfig().SecondsPerSlot * 2 / 3
	delay := time.Duration(twoThird) * time.Second

	startTime := slotutil.SlotStartTime(v.genesisTime, slot)
	finalTime := startTime.Add(delay)
	time.Sleep(roughtime.Until(finalTime))
}

func (v *validator) addIndicesToLog(ctx context.Context, committeeIndex uint64, pubKey []byte) error {
	v.attLogsLock.Lock()
	defer v.attLogsLock.Unlock()
	res, err := v.validatorClient.ValidatorIndex(ctx, &ethpb.ValidatorIndexRequest{PublicKey: pubKey})
	if err != nil {
		return err
	}

	for _, log := range v.attLogs {
		if committeeIndex == log.data.CommitteeIndex {
			log.aggregatorIndices = append(log.aggregatorIndices, res.Index)
		}
	}

	return nil
}
