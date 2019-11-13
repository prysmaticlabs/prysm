package epoch

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

// ComputeValidatorParticipation by matching validator attestations from the previous epoch,
// computing the attesting balance, and how much attested compared to the total balance.
func ComputeValidatorParticipation(state *pb.BeaconState, epoch uint64) (*ethpb.ValidatorParticipation, error) {
	currentEpoch := helpers.CurrentEpoch(state)
	previousEpoch := helpers.PrevEpoch(state)
	if epoch != currentEpoch && epoch != previousEpoch {
		return nil, fmt.Errorf(
			"requested epoch is not previous epoch %d or current epoch %d, requested %d",
			previousEpoch,
			currentEpoch,
			epoch,
		)
	}
	atts, err := MatchAttestations(state, epoch)
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve head attestations")
	}
	attestedBalances, err := AttestingBalance(state, atts.Target)
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve attested balances")
	}
	totalBalances, err := helpers.TotalActiveBalance(state)
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve total balances")
	}
	return &ethpb.ValidatorParticipation{
		GlobalParticipationRate: float32(attestedBalances) / float32(totalBalances),
		VotedEther:              attestedBalances,
		EligibleEther:           totalBalances,
	}, nil
}
