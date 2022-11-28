//go:build use_beacon_api
// +build use_beacon_api

package beacon_api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/api/gateway/apimiddleware"
	rpcmiddleware "github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
)

type forkVersionProvider interface {
	GetForkVersion(epoch types.Epoch) ([4]byte, error)
}

type beaconApiForkVersionProvider struct {
	httpClient http.Client
	url        string
}

func (c beaconApiForkVersionProvider) GetForkVersion(epoch types.Epoch) ([4]byte, error) {
	// Get the first slot of the epoch
	var forkVersion [4]byte
	firstSlotInEpoch, err := slots.EpochStart(epoch)
	if err != nil {
		return forkVersion, errors.Wrapf(err, "failed to get the fork version for epoch %d", epoch)
	}

	// Query the fork corresponding to the slot in the epoch
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/eth/v1/beacon/states/%d/fork", c.url, firstSlotInEpoch))
	if err != nil {
		return forkVersion, errors.Wrap(err, "failed to query REST API fork endpoint")
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			return
		}
	}()

	if resp.StatusCode != http.StatusOK {
		errorJson := apimiddleware.DefaultErrorJson{}
		if err := json.NewDecoder(resp.Body).Decode(&errorJson); err != nil {
			return forkVersion, errors.Wrap(err, "failed to decode response body state fork error json")
		}

		return forkVersion, errors.Errorf("error %d: %s", errorJson.Code, errorJson.Message)
	}

	stateForkJson := &rpcmiddleware.StateForkResponseJson{}
	if err := json.NewDecoder(resp.Body).Decode(&stateForkJson); err != nil {
		return forkVersion, errors.Wrap(err, "failed to decode response body state fork json")
	}

	if stateForkJson.Data == nil {
		return forkVersion, errors.New("state fork data is nil")
	}

	if !validForkVersion(stateForkJson.Data.CurrentVersion) {
		return forkVersion, errors.Errorf("invalid fork version: %s", stateForkJson.Data.CurrentVersion)
	}

	forkVersionSlice, err := hexutil.Decode(stateForkJson.Data.CurrentVersion)
	if err != nil {
		return forkVersion, errors.Wrapf(err, "failed to decode fork version: %s", stateForkJson.Data.CurrentVersion)
	}

	copy(forkVersion[:], forkVersionSlice)
	return forkVersion, nil
}
