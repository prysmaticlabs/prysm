package client

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// SubmitSlotSignature submits whether the validator's signed slot signature to the beacon node
// via gRPC. Beacon node will verify the slot signature and broadcast aggregated signature and
// proof on the validator's behave if the validator is the aggregator at the correct slot.
func (v *validator) SubmitSlotSignature(ctx context.Context, slot uint64, pubKey [48]byte) {
	ctx, span := trace.StartSpan(ctx, "validator.IsAggregator")
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

	res, err := v.aggregatorClient.SubmitSlotSignature(ctx, &pb.AggregationRequest{
		Slot:           slot,
		CommitteeIndex: assignment.CommitteeIndex,
		PublicKey:      pubKey[:],
		SlotSignature:  slotSig,
	})
	if err != nil {
		log.Errorf("Could not submit slot signature to beacon node: %v", err)
		return
	}

	if res.Aggregated {
		log.WithFields(logrus.Fields{
			"slot":           slot,
			"committeeIndex": assignment.CommitteeIndex,
			"pubKey":         fmt.Sprintf("%#x", bytesutil.Trunc(pubKey[:])),
		}).Info("Assigned as an aggregator and submitted aggregation request")
	}
}

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
