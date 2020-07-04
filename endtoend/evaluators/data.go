package evaluators

import (
	"context"
	"errors"

	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/endtoend/types"
	"google.golang.org/grpc"
)

const epochToCheck = 50 // must be more than 46 (32 hot states + 16 chkpt interval)

// ColdStateCheckpoint checks data from the database using cold state storage.
var ColdStateCheckpoint = types.Evaluator{
	Name: "cold_state_assignments_from_epoch_%d",
	Policy: func(currentEpoch uint64) bool {
		return currentEpoch == epochToCheck
	},
	Evaluation: checkColdStateCheckpoint,
}

// Checks the first node for an old checkpoint using cold state storage.
func checkColdStateCheckpoint(conns ...*grpc.ClientConn) error {
	ctx := context.Background()
	client := eth.NewBeaconChainClient(conns[0])

	for i := uint64(0); i < epochToCheck; i++ {
		res, err := client.ListValidatorAssignments(ctx, &eth.ListValidatorAssignmentsRequest{
			QueryFilter: &eth.ListValidatorAssignmentsRequest_Epoch{Epoch: i},
		})
		if err != nil {
			return err
		}
		// A simple check to ensure we received some data.
		if res == nil || res.Epoch != i {
			return errors.New("failed to return a validator assignments response for an old epoch " +
				"using cold state storage from the database")
		}
	}

	return nil
}
