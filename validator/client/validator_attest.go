package client

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"time"

	bitfield "github.com/prysmaticlabs/go-bitfield"
	ssz "github.com/prysmaticlabs/go-ssz"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
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

	tpk := hex.EncodeToString(v.keys[pk].PublicKey.Marshal())[:12]

	span.AddAttributes(
		trace.StringAttribute("validator", tpk),
	)

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
		log.WithError(err).WithFields(logrus.Fields{
			"pubKey": tpk,
		}).Error("Failed to sign attestation data and custody bit")
		return
	}
	sig := v.keys[pk].SecretKey.Sign(root[:], domain.SignatureDomain).Marshal()

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

	log.WithFields(logrus.Fields{
		"headRoot":    fmt.Sprintf("%#x", bytesutil.Trunc(data.BeaconBlockRoot)),
		"shard":       data.Crosslink.Shard,
		"sourceEpoch": data.Source.Epoch,
		"targetEpoch": data.Target.Epoch,
		"pubKey":      tpk,
	}).Info("Attested latest head")

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

	duration := time.Duration(slot*params.BeaconConfig().SecondsPerSlot+delay) * time.Second
	timeToBroadcast := time.Unix(int64(v.genesisTime), 0).Add(duration)

	time.Sleep(roughtime.Until(timeToBroadcast))
}
