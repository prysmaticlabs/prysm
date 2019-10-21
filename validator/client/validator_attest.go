package client

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"go.opencensus.io/trace"
)

// AttestToBlockHead completes the validator client's attester responsibility at a given slot.
// It fetches the latest beacon block head along with the latest canonical beacon state
// information in order to sign the block and include information about the validator's
// participation in voting on the block.
func (v *validator) AttestToBlockHead(ctx context.Context, slot uint64, pubKey [48]byte) {
	ctx, span := trace.StartSpan(ctx, "validator.AttestToBlockHead")
	defer span.End()

	span.AddAttributes(trace.StringAttribute("validator", fmt.Sprintf("%#x", pubKey)))
	log := log.WithField("pubKey", fmt.Sprintf("%#x", bytesutil.Trunc(pubKey[:])))

	// We fetch the validator index as it is necessary to generate the aggregation
	// bitfield of the attestation itself.
	var assignment *pb.AssignmentResponse_ValidatorAssignment
	if v.assignments == nil {
		log.Errorf("No assignments for validators")
		return
	}
	for _, assign := range v.assignments.ValidatorAssignment {
		if bytes.Equal(pubKey[:], assign.PublicKey) {
			assignment = assign
			break
		}
	}
	idxReq := &pb.ValidatorIndexRequest{
		PublicKey: pubKey[:],
	}
	validatorIndexRes, err := v.validatorClient.ValidatorIndex(ctx, idxReq)
	if err != nil {
		log.Errorf("Could not fetch validator index: %v", err)
		return
	}

	v.waitToSlotMidpoint(ctx, slot)

	req := &pb.AttestationRequest{
		Slot:  slot,
		Shard: assignment.Shard,
	}
	data, err := v.attesterClient.RequestAttestation(ctx, req)
	if err != nil {
		log.Errorf("Could not request attestation to sign at slot %d: %v",
			slot, err)
		return
	}

	custodyBitfield := bitfield.NewBitlist(uint64(len(assignment.Committee)))

	// Find the index in committee to be used for
	// the aggregation bitfield
	var indexInCommittee uint64
	for i, vIndex := range assignment.Committee {
		if vIndex == validatorIndexRes.Index {
			indexInCommittee = uint64(i)
			break
		}
	}

	aggregationBitfield := bitfield.NewBitlist(uint64(len(assignment.Committee)))
	aggregationBitfield.SetBitAt(indexInCommittee, true)

	domain, err := v.validatorClient.DomainData(ctx, &pb.DomainRequest{Epoch: data.Target.Epoch, Domain: params.BeaconConfig().DomainAttestation})
	if err != nil {
		log.WithError(err).Error("Failed to get domain data from beacon node")
		return
	}
	attDataAndCustodyBit := &pbp2p.AttestationDataAndCustodyBit{
		Data: data,
		// Default is false until phase 1 where proof of custody gets implemented.
		CustodyBit: false,
	}

	root, err := ssz.HashTreeRoot(attDataAndCustodyBit)
	if err != nil {
		log.WithError(err).Error("Failed to sign attestation data and custody bit")
		return
	}
	sig := v.keys[pubKey].SecretKey.Sign(root[:], domain.SignatureDomain).Marshal()

	attestation := &ethpb.Attestation{
		Data:            data,
		CustodyBits:     custodyBitfield,
		AggregationBits: aggregationBitfield,
		Signature:       sig,
	}

	attResp, err := v.attesterClient.SubmitAttestation(ctx, attestation)
	if err != nil {
		log.Errorf("Could not submit attestation to beacon node: %v", err)
		return
	}

	headRoot := fmt.Sprintf("%#x", bytesutil.Trunc(data.BeaconBlockRoot))
	log.WithField("headRoot", headRoot).Info("Submitted new attestation")

	span.AddAttributes(
		trace.Int64Attribute("slot", int64(slot)),
		trace.StringAttribute("attestationHash", fmt.Sprintf("%#x", attResp.Root)),
		trace.Int64Attribute("shard", int64(data.Crosslink.Shard)),
		trace.StringAttribute("blockRoot", fmt.Sprintf("%#x", data.BeaconBlockRoot)),
		trace.Int64Attribute("justifiedEpoch", int64(data.Source.Epoch)),
		trace.Int64Attribute("targetEpoch", int64(data.Target.Epoch)),
		trace.StringAttribute("bitfield", fmt.Sprintf("%#x", aggregationBitfield)),
	)
}

// waitToSlotMidpoint waits until halfway through the current slot period
// such that any blocks from this slot have time to reach the beacon node
// before creating the attestation.
func (v *validator) waitToSlotMidpoint(ctx context.Context, slot uint64) {
	_, span := trace.StartSpan(ctx, "validator.waitToSlotMidpoint")
	defer span.End()

	half := params.BeaconConfig().SecondsPerSlot / 2
	delay := time.Duration(half) * time.Second
	if half == 0 {
		delay = 500 * time.Millisecond
	}
	duration := time.Duration(slot*params.BeaconConfig().SecondsPerSlot) + delay
	timeToBroadcast := time.Unix(int64(v.genesisTime), 0).Add(duration)
	time.Sleep(roughtime.Until(timeToBroadcast))
}
