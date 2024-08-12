package sync

import (
	"context"
	"fmt"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"go.opencensus.io/trace"
)

// validateCommitteeIndexElectra implements the following checks from the spec:
//   - [REJECT] len(committee_indices) == 1, where committee_indices = get_committee_indices(attestation).
//   - [REJECT] attestation.data.index == 0
func validateCommitteeIndexElectra(ctx context.Context, a *ethpb.AttestationElectra) (primitives.CommitteeIndex, pubsub.ValidationResult, error) {
	_, span := trace.StartSpan(ctx, "sync.validateCommitteeIndexElectra")
	defer span.End()

	ci := a.Data.CommitteeIndex
	if ci != 0 {
		return 0, pubsub.ValidationReject, fmt.Errorf("committee index must be 0 but was %d", ci)
	}
	committeeIndices := helpers.CommitteeIndices(a.CommitteeBits)
	if len(committeeIndices) != 1 {
		return 0, pubsub.ValidationReject, fmt.Errorf("exactly 1 committee index must be set but %d were set", len(committeeIndices))
	}
	return committeeIndices[0], pubsub.ValidationAccept, nil
}
