package client

import (
	"context"
	"fmt"

	emptypb "github.com/golang/protobuf/ptypes/empty"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	validatorpb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

// SubmitSyncCommitteeMessage submits the sync committee message to the beacon chain.
func (v *validator) SubmitSyncCommitteeMessage(ctx context.Context, slot types.Slot, pubKey [48]byte) {
	ctx, span := trace.StartSpan(ctx, "validator.SubmitSyncCommitteeMessage")
	defer span.End()
	span.AddAttributes(trace.StringAttribute("validator", fmt.Sprintf("%#x", pubKey)))

	v.waitOneThirdOrValidBlock(ctx, slot)

	res, err := v.validatorClient.GetSyncMessageBlockRoot(ctx, &emptypb.Empty{})
	if err != nil {
		log.WithError(err).Error("Could not request sync message block root to sign")
		traceutil.AnnotateError(span, err)
		return
	}

	duty, err := v.duty(pubKey)
	if err != nil {
		log.WithError(err).Error("Could not fetch validator assignment")
		return
	}

	d, err := v.domainData(ctx, helpers.SlotToEpoch(slot), params.BeaconConfig().DomainSyncCommittee[:])
	if err != nil {
		log.WithError(err).Error("Could not get sync committee domain data")
		return
	}
	sszRoot := types.SSZBytes(res.Root)
	r, err := helpers.ComputeSigningRoot(&sszRoot, d.SignatureDomain)
	if err != nil {
		log.WithError(err).Error("Could not get sync committee message signing root")
		return
	}
	sig, err := v.keyManager.Sign(ctx, &validatorpb.SignRequest{
		PublicKey:       pubKey[:],
		SigningRoot:     r[:],
		SignatureDomain: d.SignatureDomain,
	})
	if err != nil {
		log.WithError(err).Error("Could not sign sync committee message")
		return
	}

	msg := &ethpb.SyncCommitteeMessage{
		Slot:           slot,
		BlockRoot:      res.Root,
		ValidatorIndex: duty.ValidatorIndex,
		Signature:      sig.Marshal(),
	}
	if _, err := v.validatorClient.SubmitSyncMessage(ctx, msg); err != nil {
		log.WithError(err).Error("Could not submit sync committee message")
		return
	}
}
