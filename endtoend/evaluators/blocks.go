package evaluators

import (
	"context"
	"fmt"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// AllAttestationsReceived -
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
		QueryFilter:          &eth.ListBlocksRequest_Epoch{Epoch: chainHead.HeadEpoch-1},
		PageSize:             int32(params.BeaconConfig().SlotsPerEpoch),
		PageToken:            "",
		XXX_NoUnkeyedLiteral: struct{}{},
		XXX_unrecognized:     nil,
		XXX_sizecache:        0,
	}
	blkResp, err := client.ListBlocks(context.Background(), blocksRequest)
	if err != nil {
		return errors.Wrap(err, "failed to get blocks")
	}


	committeeReq := &eth.ListCommitteesRequest{
		QueryFilter: &eth.ListCommitteesRequest_Epoch{Epoch: chainHead.HeadEpoch-1},
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
		blkAtts := blkContainer.Block.Block.Body.Attestations
		blockSlot := blkContainer.Block.Block.Slot
		slotCommittees := committees.Committees[blockSlot].Committees
		if blockSlot > 4 &&  len(slotCommittees) != len(blkAtts) {
			return fmt.Errorf(
				"expected block to have %d attestations, received %d for block slot %d",
				len(slotCommittees),
				len(blkAtts),
				blockSlot,
			)
		}
	}

	return nil
}
