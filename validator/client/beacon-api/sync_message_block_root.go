package beacon_api

import (
	"context"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

func (c *beaconApiValidatorClient) getSyncMessageBlockRoot(ctx context.Context) (*ethpb.SyncMessageBlockRootResponse, error) {
	// Get head beacon block root.
	var resp apimiddleware.BlockRootResponseJson
	if _, err := c.jsonRestHandler.GetRestJsonResponse(ctx, "/eth/v1/beacon/blocks/head/root", &resp); err != nil {
		return nil, errors.Wrap(err, "failed to query GET REST endpoint")
	}

	// An optimistic validator MUST NOT participate in sync committees
	// (i.e., sign across the DOMAIN_SYNC_COMMITTEE, DOMAIN_SYNC_COMMITTEE_SELECTION_PROOF or DOMAIN_CONTRIBUTION_AND_PROOF domains).
	if resp.ExecutionOptimistic {
		return nil, errors.New("the node is currently optimistic and cannot serve validators")
	}

	if resp.Data == nil {
		return nil, errors.New("no data returned")
	}

	if resp.Data.Root == "" {
		return nil, errors.New("no root returned")
	}

	blockRoot, err := hexutil.Decode(resp.Data.Root)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode beacon block root")
	}

	return &ethpb.SyncMessageBlockRootResponse{
		Root: blockRoot,
	}, nil
}
