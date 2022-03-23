package evaluators

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	coreHelper "github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	wrapperv2 "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/endtoend/helpers"
	"github.com/prysmaticlabs/prysm/testing/endtoend/policies"
	"github.com/prysmaticlabs/prysm/testing/endtoend/types"
	"github.com/prysmaticlabs/prysm/time/slots"
	"google.golang.org/grpc"
)

// AltairForkTransition ensures that the Altair hard fork has occurred successfully.
var AltairForkTransition = types.Evaluator{
	Name:       "altair_fork_transition_%d",
	Policy:     policies.OnEpoch(helpers.AltairE2EForkEpoch),
	Evaluation: altairForkOccurs,
}

// BellatrixForkTransition ensures that the Bellatrix hard fork has occurred successfully.
var BellatrixForkTransition = types.Evaluator{
	Name:       "bellatrix_fork_transition_%d",
	Policy:     policies.OnEpoch(helpers.BellatrixE2EForkEpoch),
	Evaluation: bellatrixForkOccurs,
}

func altairForkOccurs(conns ...*grpc.ClientConn) error {
	conn := conns[0]
	client := ethpb.NewBeaconNodeValidatorClient(conn)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream, err := client.StreamBlocksAltair(ctx, &ethpb.StreamBlocksRequest{VerifiedOnly: true})
	if err != nil {
		return errors.Wrap(err, "failed to get stream")
	}
	fSlot, err := slots.EpochStart(helpers.AltairE2EForkEpoch)
	if err != nil {
		return err
	}
	if ctx.Err() == context.Canceled {
		return errors.New("context canceled prematurely")
	}
	fmt.Println("Before altair receive")
	res, err := stream.Recv()
	fmt.Println("After altair receive")
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
	blk, err := wrapperv2.WrappedAltairSignedBeaconBlock(res.GetAltairBlock())
	if err != nil {
		return err
	}
	if err := coreHelper.BeaconBlockIsNil(blk); err != nil {
		return err
	}
	if blk.Block().Slot() < fSlot {
		return errors.Errorf("wanted a block >= %d but received %d", fSlot, blk.Block().Slot())
	}
	return nil
}

func bellatrixForkOccurs(conns ...*grpc.ClientConn) error {
	conn := conns[0]
	client := ethpb.NewBeaconNodeValidatorClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	stream, err := client.StreamBlocksAltair(ctx, &ethpb.StreamBlocksRequest{VerifiedOnly: true})
	if err != nil {
		return errors.Wrap(err, "failed to get stream")
	}
	fSlot, err := slots.EpochStart(helpers.BellatrixE2EForkEpoch)
	if err != nil {
		return err
	}
	if ctx.Err() == context.Canceled {
		return errors.New("context canceled prematurely")
	}
	fmt.Println("Before bellatrix receive")
	res, err := stream.Recv()
	fmt.Println("After bellatrix receive")
	if err != nil {
		return err
	}
	if res == nil || res.Block == nil {
		return errors.New("nil block returned by beacon node")
	}
	if res.GetPhase0Block() == nil && res.GetAltairBlock() == nil && res.GetBellatrixBlock() == nil {
		return errors.New("nil block returned by beacon node")
	}
	if res.GetPhase0Block() != nil {
		return errors.New("phase 0 block returned after bellatrix fork has occurred")
	}
	if res.GetAltairBlock() != nil {
		return errors.New("altair block returned after bellatrix fork has occurred")
	}
	blk, err := wrapperv2.WrappedBellatrixSignedBeaconBlock(res.GetBellatrixBlock())
	if err != nil {
		return err
	}
	if err := coreHelper.BeaconBlockIsNil(blk); err != nil {
		return err
	}
	if blk.Block().Slot() < fSlot {
		return errors.Errorf("wanted a block >= %d but received %d", fSlot, blk.Block().Slot())
	}
	return nil
}
