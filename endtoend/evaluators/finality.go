package evaluators

import (
	"context"
	"fmt"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"google.golang.org/grpc"
)

// FinalizationOccurs is an evaluator to make sure finalization is performing as it should.
// Requires to be run after at least 4 epochs have passed.
var FinalizationOccurs = Evaluator{
	Name:       "finalizes_at_epoch_%d",
	Policy:     afterNthEpoch(3),
	Evaluation: finalizationOccurs,
}

// NoFinalization is an evaluator for checking if finalization doesn't occur. Used for the slashing E2E.
var NoFinalization = Evaluator{
	Name:       "no_finalization_at_epoch_%d",
	Policy:     afterNthEpoch(0),
	Evaluation: noFinalization,
}

func finalizationOccurs(conn *grpc.ClientConn) error {
	client := eth.NewBeaconChainClient(conn)
	chainHead, err := client.GetChainHead(context.Background(), &ptypes.Empty{})
	if err != nil {
		return errors.Wrap(err, "failed to get chain head")
	}
	currentEpoch := chainHead.HeadEpoch
	finalizedEpoch := chainHead.FinalizedEpoch

	expectedFinalizedEpoch := currentEpoch - 2
	if expectedFinalizedEpoch != finalizedEpoch {
		return fmt.Errorf(
			"expected finalized epoch to be %d, received: %d",
			expectedFinalizedEpoch,
			finalizedEpoch,
		)
	}
	previousJustifiedEpoch := chainHead.PreviousJustifiedEpoch
	currentJustifiedEpoch := chainHead.JustifiedEpoch
	if previousJustifiedEpoch+1 != currentJustifiedEpoch {
		return fmt.Errorf(
			"there should be no gaps between current and previous justified epochs, received current %d and previous %d",
			currentJustifiedEpoch,
			previousJustifiedEpoch,
		)
	}
	if currentJustifiedEpoch+1 != currentEpoch {
		return fmt.Errorf(
			"there should be no gaps between current epoch and current justified epoch, received current %d and justified %d",
			currentEpoch,
			currentJustifiedEpoch,
		)
	}
	return nil
}

func noFinalization(conn *grpc.ClientConn) error {
	client := eth.NewBeaconChainClient(conn)
	chainHead, err := client.GetChainHead(context.Background(), &ptypes.Empty{})
	if err != nil {
		return errors.Wrap(err, "failed to get chain head")
	}
	finalizedEpoch := chainHead.FinalizedEpoch

	if finalizedEpoch != 0 {
		return fmt.Errorf(
			"expected finalized epoch to be 0, received: %d",
			finalizedEpoch,
		)
	}
	previousJustifiedEpoch := chainHead.PreviousJustifiedEpoch
	currentJustifiedEpoch := chainHead.JustifiedEpoch
	if currentJustifiedEpoch != 0 {
		return fmt.Errorf(
			"expected current justified epoch to be 0, received: %d",
			currentJustifiedEpoch,
		)
	}
	if previousJustifiedEpoch != 0 {
		return fmt.Errorf(
			"expected previous justified epoch to be 0, received: %d",
			previousJustifiedEpoch,
		)
	}
	return nil
}
