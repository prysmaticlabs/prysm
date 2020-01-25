package evaluators

import (
	"context"
	"fmt"
	"strings"

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
		if blockSlot == 0 {
			continue
		}
		if len(blkAtts) == 0 {
			continue
		}
		slotCommittees := committees.Committees[blockSlot-1]
		if slotCommittees == nil {
			continue
		}
		if blockSlot > 2 && len(slotCommittees.Committees) != len(blkAtts) {
			atts := []string{}
			for _, att := range blkAtts {
				atts = append(atts, fmt.Sprintf("att at slot %d, index %d, bits - %08b", att.Data.Slot, att.Data.CommitteeIndex, att.AggregationBits))
			}
			return fmt.Errorf(
				"expected block to have %d attestations, received %d for block slot %d: %s",
				len(slotCommittees.Committees),
				len(blkAtts),
				blockSlot,
				strings.Join(atts, " "),
			)
		}
	}

	return nil
}
