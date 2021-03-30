package registration

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/shared/params"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// PowchainPreregistration prepares data for powchain.Service's registration.
func PowchainPreregistration(cliCtx *cli.Context) (depositContractAddress string, endpoints []string) {
	depositContractAddress = params.BeaconConfig().DepositContractAddress
	if depositContractAddress == "" {
		log.Fatal("Valid deposit contract is required")
	}

	if !common.IsHexAddress(depositContractAddress) {
		log.Fatalf("Invalid deposit contract address given: %s", depositContractAddress)
	}

	if cliCtx.String(flags.HTTPWeb3ProviderFlag.Name) == "" {
		log.Error(
			"No ETH1 node specified to run with the beacon node. Please consider running your own ETH1 node for better uptime, security, and decentralization of ETH2. Visit https://docs.prylabs.network/docs/prysm-usage/setup-eth1 for more information.",
		)
		log.Error(
			"You will need to specify --http-web3provider to attach an eth1 node to the prysm node. Without an eth1 node block proposals for your validator will be affected and the beacon node will not be able to initialize the genesis state.",
		)
	}
	endpoints = []string{cliCtx.String(flags.HTTPWeb3ProviderFlag.Name)}
	endpoints = append(endpoints, cliCtx.StringSlice(flags.FallbackWeb3ProviderFlag.Name)...)

	return
}
