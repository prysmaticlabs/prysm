package beacon_api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

func (c *beaconApiValidatorClient) submitSyncMessage(ctx context.Context, syncMessage *ethpb.SyncCommitteeMessage) error {
	const endpoint = "/eth/v1/beacon/pool/sync_committees"

	jsonSyncCommitteeMessage := &apimiddleware.SyncCommitteeMessageJson{
		Slot:            strconv.FormatUint(uint64(syncMessage.Slot), 10),
		BeaconBlockRoot: hexutil.Encode(syncMessage.BlockRoot),
		ValidatorIndex:  strconv.FormatUint(uint64(syncMessage.ValidatorIndex), 10),
		Signature:       hexutil.Encode(syncMessage.Signature),
	}

	marshalledJsonSyncCommitteeMessage, err := json.Marshal([]*apimiddleware.SyncCommitteeMessageJson{jsonSyncCommitteeMessage})
	if err != nil {
		return errors.Wrap(err, "failed to marshal sync committee message")
	}

	if _, err := c.jsonRestHandler.PostRestJson(ctx, endpoint, nil, bytes.NewBuffer(marshalledJsonSyncCommitteeMessage), nil); err != nil {
		return errors.Wrapf(err, "failed to send POST data to `%s` REST endpoint", endpoint)
	}

	return nil
}

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

func (c *beaconApiValidatorClient) getSyncCommitteeContribution(
	ctx context.Context,
	req *ethpb.SyncCommitteeContributionRequest,
) (*ethpb.SyncCommitteeContribution, error) {
	blockRootResponse, err := c.getSyncMessageBlockRoot(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get sync message block root")
	}

	blockRoot := hexutil.Encode(blockRootResponse.Root)
	url := fmt.Sprintf("/eth/v1/validator/sync_committee_contribution?slot=%d&subcommittee_index=%d&beacon_block_root=%s",
		uint64(req.Slot), req.SubnetId, blockRoot)

	var resp apimiddleware.ProduceSyncCommitteeContributionResponseJson
	if _, err := c.jsonRestHandler.GetRestJsonResponse(ctx, url, &resp); err != nil {
		return nil, errors.Wrap(err, "failed to query GET REST endpoint")
	}

	return convertSyncContributionJsonToProto(resp.Data)
}

func convertSyncContributionJsonToProto(contribution *apimiddleware.SyncCommitteeContributionJson) (*ethpb.SyncCommitteeContribution, error) {
	if contribution == nil {
		return nil, errors.New("sync committee contribution is nil")
	}

	slot, err := strconv.ParseUint(contribution.Slot, 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse slot `%s`", contribution.Slot)
	}

	blockRoot, err := hexutil.Decode(contribution.BeaconBlockRoot)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to beacon block root `%s`", contribution.BeaconBlockRoot)
	}

	subcommitteeIdx, err := strconv.ParseUint(contribution.SubcommitteeIndex, 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse subcommittee index `%s`", contribution.SubcommitteeIndex)
	}

	aggregationBits, err := hexutil.Decode(contribution.AggregationBits)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode aggregation bits `%s`", contribution.AggregationBits)
	}

	signature, err := hexutil.Decode(contribution.Signature)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode contribution signature `%s`", contribution.Signature)
	}

	return &ethpb.SyncCommitteeContribution{
		Slot:              types.Slot(slot),
		BlockRoot:         blockRoot,
		SubcommitteeIndex: subcommitteeIdx,
		AggregationBits:   aggregationBits,
		Signature:         signature,
	}, nil
}
