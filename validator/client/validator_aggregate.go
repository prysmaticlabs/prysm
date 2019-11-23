package client

import (
	"context"
	"fmt"
	"time"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
	"github.com/sirupsen/logrus"
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

	assignment, err := v.assignment(pubKey)
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

	res, err := v.aggregatorClient.SubmitAggregateAndProof(ctx, &pb.AggregationRequest{
		Slot:           slot,
		CommitteeIndex: assignment.CommitteeIndex,
		PublicKey:      pubKey[:],
		SlotSignature:  slotSig,
	})
	if err != nil {
		log.Errorf("Could not submit slot signature to beacon node: %v", err)
		return
	}

	log.WithFields(logrus.Fields{
		"slot":            slot,
		"committeeIndex":  assignment.CommitteeIndex,
		"pubKey":          fmt.Sprintf("%#x", bytesutil.Trunc(pubKey[:])),
		"aggregationRoot": fmt.Sprintf("%#x", bytesutil.Trunc(res.Root[:])),
	}).Debug("Assigned and submitted aggregation and proof request")
}

// This implements selection logic outlined in:
// https://github.com/ethereum/eth2.0-specs/blob/v0.9.0/specs/validator/0_beacon-chain-validator.md#aggregation-selection
func (v *validator) signSlot(ctx context.Context, pubKey [48]byte, slot uint64) ([]byte, error) {
	domain, err := v.validatorClient.DomainData(ctx, &pb.DomainRequest{Epoch: helpers.SlotToEpoch(slot), Domain: params.BeaconConfig().DomainBeaconAttester})
	if err != nil {
		return nil, err
	}

	slotRoot, err := ssz.HashTreeRoot(slot)
	if err != nil {
		return nil, err
	}

	sig := v.keys[pubKey].SecretKey.Sign(slotRoot[:], domain.SignatureDomain)
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
