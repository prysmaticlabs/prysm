package beacon_api

import (
	"bytes"
	"encoding/json"
	"fmt"
	neturl "net/url"
	"regexp"
	"strconv"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"

	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

var beaconAPITogRPCValidatorStatus = map[string]ethpb.ValidatorStatus{
	"pending_initialized": ethpb.ValidatorStatus_DEPOSITED,
	"pending_queued":      ethpb.ValidatorStatus_PENDING,
	"active_ongoing":      ethpb.ValidatorStatus_ACTIVE,
	"active_exiting":      ethpb.ValidatorStatus_EXITING,
	"active_slashed":      ethpb.ValidatorStatus_SLASHING,
	"exited_unslashed":    ethpb.ValidatorStatus_EXITED,
	"exited_slashed":      ethpb.ValidatorStatus_EXITED,
	"withdrawal_possible": ethpb.ValidatorStatus_EXITED,
	"withdrawal_done":     ethpb.ValidatorStatus_EXITED,
}

func validRoot(root string) bool {
	matchesRegex, err := regexp.MatchString("^0x[a-fA-F0-9]{64}$", root)
	if err != nil {
		return false
	}
	return matchesRegex
}

func uint64ToString[T uint64 | types.Slot | types.ValidatorIndex | types.CommitteeIndex | types.Epoch](val T) string {
	return strconv.FormatUint(uint64(val), 10)
}

func buildURL(path string, queryParams ...neturl.Values) string {
	if len(queryParams) == 0 {
		return path
	}

	return fmt.Sprintf("%s?%s", path, queryParams[0].Encode())
}

func (c *beaconApiValidatorClient) getFork() (*apimiddleware.StateForkResponseJson, error) {
	const endpoint = "/eth/v1/beacon/states/head/fork"

	stateForkResponseJson := &apimiddleware.StateForkResponseJson{}

	_, err := c.jsonRestHandler.GetRestJsonResponse(
		endpoint,
		stateForkResponseJson,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get json response from `%s` REST endpoint", endpoint)
	}

	return stateForkResponseJson, nil
}

func (c *beaconApiValidatorClient) getHeaders() (*apimiddleware.BlockHeadersResponseJson, error) {
	const endpoint = "/eth/v1/beacon/headers"

	blockHeadersResponseJson := &apimiddleware.BlockHeadersResponseJson{}

	_, err := c.jsonRestHandler.GetRestJsonResponse(
		endpoint,
		blockHeadersResponseJson,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get json response from `%s` REST endpoint", endpoint)
	}

	return blockHeadersResponseJson, nil
}

func (c *beaconApiValidatorClient) getLiveness(epoch types.Epoch, validatorIndexes []string) (*apimiddleware.LivenessResponseJson, error) {
	const endpoint = "/eth/v1/validator/liveness/"
	url := endpoint + strconv.FormatUint(uint64(epoch), 10)

	livenessResponseJson := &apimiddleware.LivenessResponseJson{}

	marshalledJsonValidatorIndexes, err := json.Marshal(validatorIndexes)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal validator indexes")
	}

	if _, err := c.jsonRestHandler.PostRestJson(url, nil, bytes.NewBuffer(marshalledJsonValidatorIndexes), livenessResponseJson); err != nil {
		return nil, errors.Wrapf(err, "failed to send POST data to `%s` REST URL", url)
	}

	return livenessResponseJson, nil
}
