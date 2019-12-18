package sync

import (
	"context"
	"errors"

	"github.com/gogo/protobuf/proto"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
)

// Validation
// - The attestation's committee index (attestation.data.index) is for the correct subnet.
// - The attestation is unaggregated -- that is, it has exactly one participating validator (len([bit for bit in attestation.aggregation_bits if bit == 0b1]) == 1).
// - The block being voted for (attestation.data.beacon_block_root) passes validation.
// - attestation.data.slot is within the last ATTESTATION_PROPAGATION_SLOT_RANGE slots (attestation.data.slot + ATTESTATION_PROPAGATION_SLOT_RANGE >= current_slot >= attestation.data.slot).
// - The signature of attestation is valid.
func (s *Service) validateCommitteeIndexBeaconAttestation(ctx context.Context, msg proto.Message, broadcaster p2p.Broadcaster, fromSelf bool) (bool, error) {
	att, ok := msg.(*eth.Attestation)
	if !ok {
		return false, errors.New("wrong message type")
	}

	// Attestation must be unaggregated.
	if att.AggregationBits.Count() != 1 {
		return false, nil
	}

	// Attestation's block must exist in database (only valid blocks are stored).

	// Attestation's slot is within ATTESTATION_PROPAGATION_SLOT_RANGE.

	// Attestation's signature is a valid BLS signature.

	return true, nil
}
