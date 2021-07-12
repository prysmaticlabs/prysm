package evaluators

import (
	"context"

	"github.com/pkg/errors"
	eth2types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/endtoend/policies"
	"github.com/prysmaticlabs/prysm/endtoend/types"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	prysmv2 "github.com/prysmaticlabs/prysm/proto/prysm/v2"
	wrapperv2 "github.com/prysmaticlabs/prysm/proto/prysm/v2/wrapper"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/grpc"
)

// ForkTransition ensures that the hard fork has occurred successfully.
var ForkTransition = types.Evaluator{
	Name:       "fork_transition_%d",
	Policy:     policies.OnEpoch(params.AltairE2EForkEpoch),
	Evaluation: forkOccurs,
}

func forkOccurs(conns ...*grpc.ClientConn) error {
	conn := conns[0]
	client := prysmv2.NewBeaconNodeValidatorAltairClient(conn)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream, err := client.StreamBlocks(ctx, &ethpb.StreamBlocksRequest{VerifiedOnly: true})
	if err != nil {
		return errors.Wrap(err, "failed to get stream")
	}
	blockInEpoch := 0
	blockMax := 10
	fSlot, err := helpers.StartSlot(params.AltairE2EForkEpoch)
	if err != nil {
		return err
	}
	for {
		if ctx.Err() == context.Canceled {
			return errors.New("context canceled prematurely")
		}
		res, err := stream.Recv()
		if err != nil {
			return err
		}
		if res == nil || res.Block == nil {
			return errors.New("nil block returned by beacon node")
		}
		if res.GetPhase0Block() == nil && res.GetAltairBlock() == nil {
			return errors.New("nil block returned by beacon node")
		}
		if res.GetPhase0Block() != nil {
			return errors.New("phase 0 block returned after altair fork has occurred")
		}
		blk := wrapperv2.WrappedAltairSignedBeaconBlock(res.GetAltairBlock())
		if blk == nil || blk.IsNil() {
			return errors.New("nil altair block received from stream")
		}
		if blk.Block().Slot() != fSlot+eth2types.Slot(blockInEpoch) {
			return errors.Errorf("wanted a block slot of %d but received %d", fSlot+eth2types.Slot(blockInEpoch), blk.Block().Slot())
		}
		blockInEpoch++
		// Exit if we have already validated for the past 10 blocks.
		if blockInEpoch > blockMax {
			break
		}
	}
	return nil
}
