package sync

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/gogo/protobuf/proto"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

// Validation
// - The attestation's committee index (attestation.data.index) is for the correct subnet.
// - The attestation is unaggregated -- that is, it has exactly one participating validator (len([bit for bit in attestation.aggregation_bits if bit == 0b1]) == 1).
// - The block being voted for (attestation.data.beacon_block_root) passes validation.
// - attestation.data.slot is within the last ATTESTATION_PROPAGATION_SLOT_RANGE slots (attestation.data.slot + ATTESTATION_PROPAGATION_SLOT_RANGE >= current_slot >= attestation.data.slot).
// - The signature of attestation is valid.
func (s *Service) validateCommitteeIndexBeaconAttestation(ctx context.Context, msg proto.Message, broadcaster p2p.Broadcaster, fromSelf bool) (bool, error) {
	// Attestation processing requires the target block to be present in the database, so we'll skip
	// validating or processing attestations until fully synced.
	if s.initialSync.Syncing() {
		return false, nil
	}
	ctx, span := trace.StartSpan(ctx, "sync.validateCommitteeIndexBeaconAttestation")
	defer span.End()

	att, ok := msg.(*eth.Attestation)
	if !ok {
		return false, errors.New("wrong message type")
	}

	// The attestation's committee index (attestation.data.index) is for the correct subnet.
	topic := ctx.Value("topic")
	format := p2p.GossipTypeMapping[reflect.TypeOf(att)]
	if strings.HasPrefix(topic.(string), fmt.Sprintf(format, att.Data.CommitteeIndex)) {
		return false, errors.New("attestation from wrong subnet")
	}

	// Attestation must be unaggregated.
	if att.AggregationBits.Count() != 1 {
		return false, nil
	}

	// Attestation's block must exist in database (only valid blocks are stored).
	if !s.db.HasBlock(ctx, bytesutil.ToBytes32(att.Data.BeaconBlockRoot)) {
		log.WithField(
			"blockRoot",
			fmt.Sprintf("%#x", att.Data.BeaconBlockRoot),
		).WithError(errPointsToBlockNotInDatabase).Debug("Ignored incoming attestation that points to a block which is not in the database")
		traceutil.AnnotateError(span, errPointsToBlockNotInDatabase)
		return false, nil
	}

	// Attestation's slot is within ATTESTATION_PROPAGATION_SLOT_RANGE.
	currentSlot := helpers.SlotsSinceGenesis(s.chain.GenesisTime())
	upper := att.Data.Slot + params.BeaconConfig().AttestationPropagationSlotRange
	lower := att.Data.Slot
	if currentSlot > upper || currentSlot < lower {
		return false, errors.New("attestation not within propagation slot range")
	}

	// Attestation's signature is a valid BLS signature.
	if _, err := bls.SignatureFromBytes(att.Signature); err == nil {
		return false, errors.New("attestation signature is invalid")
	}

	s.p2p.Broadcast(ctx, att)
	return true, nil
}
