package evaluators

import (
	"context"
	"errors"

	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/endtoend/types"
	"google.golang.org/grpc"
)

// ColdStateCheckpoint checks data from the database using cold state storage.
var ColdStateCheckpoint = types.Evaluator{
	Name: "cold_state_assignments_for_epoch_5_from_epoch_%d",
	Policy: func(currentEpoch uint64) bool {
		return currentEpoch == 50 // must be at least 32
	},
	Evaluation: checkColdStateCheckpoint,
}

// Checks the first node for an old checkpoint using cold state storage.
func checkColdStateCheckpoint(conns ...*grpc.ClientConn) error {
	client := eth.NewBeaconChainClient(conns[0])
	// This request for epoch 24 should use the cold state checkpoint from epoch 16.
	res, err := client.ListValidatorAssignments(context.Background(), &eth.ListValidatorAssignmentsRequest{
		QueryFilter: &eth.ListValidatorAssignmentsRequest_Epoch{Epoch: 24},
	})
	if err != nil {
		return err
	}
	if res.Epoch != 24 {
		return errors.New("failed to return a validator assignments response for an old epoch " +
			"using cold state storage from the database")
	}

	return nil
}
