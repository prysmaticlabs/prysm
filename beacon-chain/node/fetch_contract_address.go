package node

import (
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/prysmaticlabs/prysm/shared/params"
)

var cachedDepositAddress string
var fetchLock sync.Mutex

// fetchDepositContract from the cluster endpoint.
func fetchDepositContract() (string, error) {
	fetchLock.Lock()
	defer fetchLock.Unlock()

	if cachedDepositAddress != "" {
		return cachedDepositAddress, nil
	}

	log.WithField(
		"endpoint",
		params.BeaconConfig().TestnetContractEndpoint,
	).Info("Fetching testnet cluster address")
	resp, err := http.Get(params.BeaconConfig().TestnetContractEndpoint)
	if err != nil {
		return "", err
	}
	contractResponse, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if err := resp.Body.Close(); err != nil {
		return "", err
	}

	cachedDepositAddress = string(contractResponse)
	return cachedDepositAddress, nil
}
