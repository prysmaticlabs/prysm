package evaluators

import (
	"context"
	"fmt"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// AllAttestationsReceived is an evaluator for ensuring all the blocks in the beacon chain
// contain the attestations needed, if a slot is skipped, those attestations will be expected
// in the next block.
var AllAttestationsReceived = Evaluator{
	Name:       "att_attestations_received_%d",
	Policy:     afterNthEpoch(1),
	Evaluation: allAttestationsReceived,
}

func allAttestationsReceived(client eth.BeaconChainClient) error {
	chainHead, err := client.GetChainHead(context.Background(), &ptypes.Empty{})
	if err != nil {
		return errors.Wrap(err, "failed to get chain head")
	}

	blocksRequest := &eth.ListBlocksRequest{
		QueryFilter: &eth.ListBlocksRequest_Epoch{Epoch: chainHead.HeadEpoch - 1},
		PageSize:    int32(params.BeaconConfig().SlotsPerEpoch),
	}
	blkResp, err := client.ListBlocks(context.Background(), blocksRequest)
	if err != nil {
		return errors.Wrap(err, "failed to get blocks")
	}

	committeeReq := &eth.ListCommitteesRequest{
		QueryFilter: &eth.ListCommitteesRequest_Epoch{Epoch: chainHead.HeadEpoch - 1},
	}
	if chainHead.HeadEpoch-1 == 0 {
		genFilter := &eth.ListCommitteesRequest_Genesis{Genesis: true}
		committeeReq.QueryFilter = genFilter
	}
	committees, err := client.ListBeaconCommittees(context.Background(), committeeReq)
	if err != nil {
		return errors.Wrap(err, "failed to get committees")
	}

	for _, blkContainer := range blkResp.BlockContainers {
		blockSlot := blkContainer.Block.Block.Slot
		slotCommittees := committees.Committees[blockSlot-1]
		blkAtts := blkContainer.Block.Block.Body.Attestations
		if blockSlot < 1 || slotCommittees == nil {
			continue
		}

		expectedAtts := len(slotCommittees.Committees)
		// If the amount of atts are not equal already, expect double the attestations.
		// This is to account for skipped slots.
		if expectedAtts != len(blkAtts) {
			expectedAtts *= 2
		}
		if expectedAtts != len(blkAtts) {
			return fmt.Errorf(
				"expected block slot %d to have %d attestations, received %d",
				blockSlot,
				expectedAtts,
				len(blkAtts),
			)
		}
	}

	return nil
}
