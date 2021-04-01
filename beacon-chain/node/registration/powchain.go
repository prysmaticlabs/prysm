package registration

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/shared/params"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// PowchainPreregistration prepares data for powchain.Service's registration.
func PowchainPreregistration(cliCtx *cli.Context) (depositContractAddress string, endpoints []string, err error) {
	depositContractAddress, err = DepositContractAddress(cliCtx)
	if err != nil {
		return "", nil, err
	}
	endpoints = []string{cliCtx.String(flags.HTTPWeb3ProviderFlag.Name)}
	endpoints = append(endpoints, cliCtx.StringSlice(flags.FallbackWeb3ProviderFlag.Name)...)
	return
}

// DepositContractAddress returns the address of the deposit contract.
func DepositContractAddress(cliCtx *cli.Context) (string, error) {
	address := params.BeaconConfig().DepositContractAddress
	if address == "" {
		return "", errors.New("valid deposit contract is required")
	}

	if !common.IsHexAddress(address) {
		return "", errors.New("invalid deposit contract address given: " + address)
	}

	if cliCtx.String(flags.HTTPWeb3ProviderFlag.Name) == "" {
		log.Error(
			"No ETH1 node specified to run with the beacon node. Please consider running your own ETH1 node for better uptime, security, and decentralization of ETH2. Visit https://docs.prylabs.network/docs/prysm-usage/setup-eth1 for more information.",
		)
		log.Error(
			"You will need to specify --http-web3provider to attach an eth1 node to the prysm node. Without an eth1 node block proposals for your validator will be affected and the beacon node will not be able to initialize the genesis state.",
		)
	}

	return address, nil
}
