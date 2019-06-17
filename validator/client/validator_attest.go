package client

import (
	"bytes"
	"context"
	"fmt"
	"time"

	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var delay = params.BeaconConfig().SecondsPerSlot / 2

// AttestToBlockHead completes the validator client's attester responsibility at a given slot.
// It fetches the latest beacon block head along with the latest canonical beacon state
// information in order to sign the block and include information about the validator's
// participation in voting on the block.
func (v *validator) AttestToBlockHead(ctx context.Context, slot uint64, pk string) {
	ctx, span := trace.StartSpan(ctx, "validator.AttestToBlockHead")
	defer span.End()
	span.AddAttributes(
		trace.StringAttribute("validator", fmt.Sprintf("%#x", v.keys[pk].PublicKey.Marshal())),
	)
	truncatedPk := bytesutil.Trunc([]byte(pk))

	log.WithField("validator", truncatedPk).Info("Attesting to a beacon block")
	v.waitToSlotMidpoint(ctx, slot)

	// We fetch the validator index as it is necessary to generate the aggregation
	// bitfield of the attestation itself.
	pubKey := v.keys[pk].PublicKey.Marshal()
	var assignment *pb.AssignmentResponse_ValidatorAssignment
	if v.assignments == nil {
		log.Errorf("No assignments for validators")
		return
	}
	for _, assign := range v.assignments.ValidatorAssignment {
		if bytes.Equal(pubKey, assign.PublicKey) {
			assignment = assign
			break
		}
	}
	idxReq := &pb.ValidatorIndexRequest{
		PublicKey: pubKey,
	}
	validatorIndexRes, err := v.validatorClient.ValidatorIndex(ctx, idxReq)
	if err != nil {
		log.Errorf("Could not fetch validator index: %v", err)
		return
	}
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
	committeeLength := mathutil.CeilDiv8(len(assignment.Committee))

	// We set the custody bitfield to an slice of zero values as a stub for phase 0
	// of length len(committee)+7 // 8.
	custodyBitfield := make([]byte, committeeLength)

	// Find the index in committee to be used for
	// the aggregation bitfield
	var indexInCommittee int
	for i, vIndex := range assignment.Committee {
		if vIndex == validatorIndexRes.Index {
			indexInCommittee = i
			break
		}
	}
	aggregationBitfield, err := bitutil.SetBitfield(indexInCommittee, len(assignment.Committee))
	if err != nil {
		log.Errorf("Could not set bitfield: %v", err)
	}

	// TODO(#1366): Use BLS to generate an aggregate signature.
	sig := []byte("signed")

	log.WithFields(logrus.Fields{
		"shard":     data.Crosslink.Shard,
		"slot":      slot,
		"validator": truncatedPk,
	}).Info("Attesting to beacon chain head...")

	attestation := &pbp2p.Attestation{
		Data:                data,
		CustodyBitfield:     custodyBitfield,
		AggregationBitfield: aggregationBitfield,
		Signature:           sig,
	}

	attResp, err := v.attesterClient.SubmitAttestation(ctx, attestation)
	if err != nil {
		log.Errorf("Could not submit attestation to beacon node: %v", err)
		return
	}

	log.WithFields(logrus.Fields{
		"headRoot":    fmt.Sprintf("%#x", bytesutil.Trunc(data.BeaconBlockRoot)),
		"shard":       data.Crosslink.Shard,
		"sourceEpoch": data.SourceEpoch,
		"targetEpoch": data.TargetEpoch,
		"validator":   truncatedPk,
	}).Info("Attested latest head")

	span.AddAttributes(
		trace.Int64Attribute("slot", int64(slot)),
		trace.StringAttribute("attestationHash", fmt.Sprintf("%#x", attResp.Root)),
		trace.Int64Attribute("shard", int64(data.Crosslink.Shard)),
		trace.StringAttribute("blockRoot", fmt.Sprintf("%#x", data.BeaconBlockRoot)),
		trace.Int64Attribute("justifiedEpoch", int64(data.SourceEpoch)),
		trace.Int64Attribute("targetEpoch", int64(data.TargetEpoch)),
		trace.StringAttribute("bitfield", fmt.Sprintf("%#x", aggregationBitfield)),
	)
}

// waitToSlotMidpoint waits until halfway through the current slot period
// such that any blocks from this slot have time to reach the beacon node
// before creating the attestation.
func (v *validator) waitToSlotMidpoint(ctx context.Context, slot uint64) {
	_, span := trace.StartSpan(ctx, "validator.waitToSlotMidpoint")
	defer span.End()

	duration := time.Duration(slot*params.BeaconConfig().SecondsPerSlot+delay) * time.Second
	timeToBroadcast := time.Unix(int64(v.genesisTime), 0).Add(duration)

	time.Sleep(time.Until(timeToBroadcast))
}
