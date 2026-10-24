package operations

import (
	"context"
	"testing"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/eip8453"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/state"
	"github.com/OffchainLabs/prysm/v7/consensus-types/interfaces"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
)

// RunHeadVoteTest executes "operations/head_vote" tests.
func RunHeadVoteTest(t *testing.T, config string) {
	runOperationTest(t, config, "head_vote", func(b []byte) (*ethpb.BeaconBlockBodyEIP8453, error) {
		vote := &ethpb.HeadVote{}
		if err := vote.UnmarshalSSZ(b); err != nil {
			return nil, err
		}
		return &ethpb.BeaconBlockBodyEIP8453{HeadVotes: []*ethpb.HeadVote{vote}}, nil
	}, func(ctx context.Context, s state.BeaconState, body interfaces.ReadOnlyBeaconBlockBody) error {
		votes, err := body.HeadVotes()
		if err != nil {
			return err
		}
		return eip8453.ProcessHeadVote(ctx, s, votes[0])
	})
}

// RunHeightEventsTest executes "operations/height_events" tests, which carry no operation input.
func RunHeightEventsTest(t *testing.T, config string) {
	runOperationTest(t, config, "height_events", nil, func(ctx context.Context, s state.BeaconState, _ interfaces.ReadOnlyBeaconBlockBody) error {
		return eip8453.ProcessHeightEvents(ctx, s)
	})
}
