package decoupled

import (
	"fmt"

	"github.com/OffchainLabs/go-bitfield"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/state"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
)

// ValidateAvailableAttestations checks a block's AC lists: previous-slot votes only, one support bit per aggregate, at most two votes per validator.
func ValidateAvailableAttestations(st state.ReadOnlyBeaconState, blockSlot primitives.Slot, atts []*ethpb.AvailableAttestation, support bitfield.Bitlist) error {
	n := uint64(st.NumValidators())
	if uint64(len(atts)) > 2*n {
		return fmt.Errorf("%d available attestations exceed the cap of %d", len(atts), 2*n)
	}
	if support.Len() != uint64(len(atts)) {
		return fmt.Errorf("%d support votes for %d available attestations", support.Len(), len(atts))
	}
	for _, a := range atts {
		if a.Data == nil {
			return errNilAttestationData
		}
		if a.Data.Slot+1 != blockSlot {
			return fmt.Errorf("available attestation for slot %d in block %d", a.Data.Slot, blockSlot)
		}
		if a.AggregationBits.Len() > n {
			return fmt.Errorf("aggregation bits length %d beyond %d validators", a.AggregationBits.Len(), n)
		}
	}
	return nil
}
