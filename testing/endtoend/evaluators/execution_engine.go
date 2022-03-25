package evaluators

import (
	"context"
	"math"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/endtoend/helpers"
	e2e "github.com/prysmaticlabs/prysm/testing/endtoend/params"
	"github.com/prysmaticlabs/prysm/testing/endtoend/policies"
	"github.com/prysmaticlabs/prysm/testing/endtoend/types"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

// TransactionsPresent is an evaluator to make sure transactions send to the execution engine
// appear in consensus client blocks' execution payload.
var TransactionsPresent = types.Evaluator{
	Name:       "transactions_present_at_epoch_%d",
	Policy:     policies.AfterNthEpoch(helpers.BellatrixE2EForkEpoch),
	Evaluation: transactionsPresent,
}

func transactionsPresent(conns ...*grpc.ClientConn) error {
	conn := conns[0]
	client := ethpb.NewBeaconChainClient(conn)
	chainHead, err := client.GetChainHead(context.Background(), &emptypb.Empty{})
	if err != nil {
		return errors.Wrap(err, "failed to get chain head")
	}
	req := &ethpb.ListBlocksRequest{QueryFilter: &ethpb.ListBlocksRequest_Epoch{Epoch: chainHead.HeadEpoch.Sub(1)}}
	blks, err := client.ListBeaconBlocks(context.Background(), req)
	if err != nil {
		return errors.Wrap(err, "failed to get blocks from beacon-chain")
	}
	expectedTxNum := int(math.Round(float64(params.E2ETestConfig().SlotsPerEpoch) * float64(e2e.NumOfExecEngineTxs) * 0.7))
	var numberOfTxs int
	for _, ctr := range blks.BlockContainers {
		switch ctr.Block.(type) {
		case *ethpb.BeaconBlockContainer_BellatrixBlock:
			numberOfTxs += len(ctr.GetBellatrixBlock().Block.Body.ExecutionPayload.Transactions)
		}
	}
	if numberOfTxs < expectedTxNum {
		return errors.Errorf(
			"not enough transactions in execution payload, expected=%d vs actual=%d",
			expectedTxNum,
			numberOfTxs,
		)
	}
	return nil
}
