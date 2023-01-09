package beacon_api

import (
	"bytes"
	"context"
	"encoding/json"
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
