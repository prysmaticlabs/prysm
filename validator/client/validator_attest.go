package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
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

	log := log.WithField("pubKey", fmt.Sprintf("%#x", bytesutil.Trunc(pubKey[:]))).WithField("slot", slot)
	duty, err := v.duty(pubKey)
	if err != nil {
		log.WithError(err).Error("Could not fetch validator assignment")
		return
	}

	indexInCommittee, validatorIndex, err := v.indexInCommittee(pubKey, duty)
	if err != nil {
		log.WithError(err).Error("Could not get validator index in assignment")
		return
	}

	// As specified in the spec, an attester should wait until one-third of the way through the slot,
	// then create and broadcast the attestation.
	// https://github.com/ethereum/eth2.0-specs/blob/v0.9.3/specs/validator/0_beacon-chain-validator.md#attesting
	v.waitToOneThird(ctx, slot)

	req := &ethpb.AttestationDataRequest{
		Slot:           slot,
		CommitteeIndex: duty.CommitteeIndex,
	}
	data, err := v.validatorClient.GetAttestationData(ctx, req)
	if err != nil {
		log.WithError(err).Error("Could not request attestation to sign at slot")
		return
	}

	sig, err := v.signAtt(ctx, pubKey, data)
	if err != nil {
		log.WithError(err).Error("Could not sign attestation")
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
		return
	}

	if err := v.saveAttesterIndexToData(data, validatorIndex); err != nil {
		log.WithError(err).Error("Could not save validator index for logging")
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
}

// waitToOneThird waits until one-third of the way through the slot
// such that any blocks from this slot have time to reach the beacon node
// before creating the attestation.
func (v *validator) waitToOneThird(ctx context.Context, slot uint64) {
	_, span := trace.StartSpan(ctx, "validator.waitToOneThird")
	defer span.End()

	oneThird := params.BeaconConfig().SecondsPerSlot / 3
	delay := time.Duration(oneThird) * time.Second
	if oneThird == 0 {
		delay = 500 * time.Millisecond
	}
	startTime := slotutil.SlotStartTime(v.genesisTime, slot)
	timeToBroadcast := startTime.Add(delay)
	time.Sleep(roughtime.Until(timeToBroadcast))
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

// This returns the index of validator's position in a committee. It's used to construct aggregation and
// custody bit fields.
func (v *validator) indexInCommittee(pubKey [48]byte, duty *ethpb.DutiesResponse_Duty) (uint64, uint64, error) {
	v.pubKeyToIDLock.RLock()
	defer v.pubKeyToIDLock.RUnlock()

	index := v.pubKeyToID[pubKey]
	for i, validatorIndex := range duty.Committee {
		if validatorIndex == index {
			return uint64(i), index, nil
		}
	}

	return 0, 0, fmt.Errorf("index %d not in committee", index)
}

// Given validator's public key, this returns the signature of an attestation data.
func (v *validator) signAtt(ctx context.Context, pubKey [48]byte, data *ethpb.AttestationData) ([]byte, error) {
	domain, err := v.validatorClient.DomainData(ctx, &ethpb.DomainRequest{
		Epoch:  data.Target.Epoch,
		Domain: params.BeaconConfig().DomainBeaconAttester,
	})
	if err != nil {
		return nil, err
	}

	root, err := ssz.HashTreeRoot(data)
	if err != nil {
		return nil, err
	}

	sig, err := v.keyManager.Sign(pubKey, root, domain.SignatureDomain)
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
