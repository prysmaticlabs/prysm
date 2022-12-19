package evaluators

import (
	"context"
	"strconv"

	"github.com/pkg/errors"
	ctypes "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	mathutil "github.com/prysmaticlabs/prysm/v3/math"
	"github.com/prysmaticlabs/prysm/v3/proto/eth/service"
	v2 "github.com/prysmaticlabs/prysm/v3/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/policies"
	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/types"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	"google.golang.org/grpc"
)

// OptimisticSyncEnabled checks that the node is in an optimistic state.
var OptimisticSyncEnabled = types.Evaluator{
	Name:       "optimistic_sync_at_epoch_%d",
	Policy:     policies.AllEpochs,
	Evaluation: optimisticSyncEnabled,
}

func optimisticSyncEnabled(_ types.EvaluationContext, conns ...*grpc.ClientConn) error {
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
