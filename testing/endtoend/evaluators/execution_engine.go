package evaluators

import (
	"context"
	"math"
	"strconv"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	ctypes "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	mathutil "github.com/prysmaticlabs/prysm/v3/math"
	"github.com/prysmaticlabs/prysm/v3/proto/eth/service"
	v2 "github.com/prysmaticlabs/prysm/v3/proto/eth/v2"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/helpers"
	e2e "github.com/prysmaticlabs/prysm/v3/testing/endtoend/params"
	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/policies"
	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/types"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
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

// OptimisticSyncEnabled checks that the node is in an optimistic state.
var OptimisticSyncEnabled = types.Evaluator{
	Name:       "optimistic_sync_at_epoch_%d",
	Policy:     policies.AllEpochs,
	Evaluation: optimisticSyncEnabled,
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
	expectedTxNum := int(math.Round(float64(params.E2ETestConfig().SlotsPerEpoch) * float64(e2e.NumOfExecEngineTxs) * e2e.ExpectedExecEngineTxsThreshold))
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

func optimisticSyncEnabled(conns ...*grpc.ClientConn) error {
	for _, conn := range conns {
		client := service.NewBeaconChainClient(conn)
		head, err := client.GetBlockV2(context.Background(), &v2.BlockRequestV2{BlockId: []byte("head")})
		if err != nil {
			return err
		}
		headSlot := uint64(0)
		switch hb := head.Data.Message.(type) {
		case *v2.SignedBeaconBlockContainer_Phase0Block:
			headSlot = uint64(hb.Phase0Block.Slot)
		case *v2.SignedBeaconBlockContainer_AltairBlock:
			headSlot = uint64(hb.AltairBlock.Slot)
		case *v2.SignedBeaconBlockContainer_BellatrixBlock:
			headSlot = uint64(hb.BellatrixBlock.Slot)
		default:
			return errors.New("no valid block type retrieved")
		}
		currEpoch := slots.ToEpoch(ctypes.Slot(headSlot))
		startSlot, err := slots.EpochStart(currEpoch)
		if err != nil {
			return err
		}
		isOptimistic := false
		for i := startSlot; i <= ctypes.Slot(headSlot); i++ {
			castI, err := mathutil.Int(uint64(i))
			if err != nil {
				return err
			}
			block, err := client.GetBlockV2(context.Background(), &v2.BlockRequestV2{BlockId: []byte(strconv.Itoa(castI))})
			if err != nil {
				// Continue in the event of non-existent blocks.
				continue
			}
			if !block.ExecutionOptimistic {
				return errors.New("expected block to be optimistic, but it is not")
			}
			isOptimistic = true
		}
		if !isOptimistic {
			return errors.New("expected block to be optimistic, but it is not")
		}
	}
	return nil
}
