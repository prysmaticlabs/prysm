package powchaincmd

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/cmd/beacon-chain/flags"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var log = logrus.WithField("prefix", "cmd-powchain")

// FlagOptions for powchain service flag configurations.
func FlagOptions(c *cli.Context) ([]powchain.Option, error) {
	endpoints := parseHttpEndpoints(c)
	opts := []powchain.Option{
		powchain.WithHttpEndpoints(endpoints),
		powchain.WithEth1HeaderRequestLimit(c.Uint64(flags.Eth1HeaderReqLimit.Name)),
		powchain.WithBuilderProposerEndpoint(c.String(flags.PayloadBuilderProviderFlag.Name)),
	}
	return opts, nil
}

func parseHttpEndpoints(c *cli.Context) []string {
	if c.String(flags.HTTPWeb3ProviderFlag.Name) == "" && len(c.StringSlice(flags.FallbackWeb3ProviderFlag.Name)) == 0 {
		log.Error(
			"No ETH1 node specified to run with the beacon node. " +
				"Please consider running your own Ethereum proof-of-work node for better uptime, " +
				"security, and decentralization of Ethereum. Visit " +
				"https://docs.prylabs.network/docs/prysm-usage/setup-eth1 for more information",
		)
		log.Error(
			"You will need to specify --http-web3provider and/or --fallback-web3provider to attach " +
				"an eth1 node to the prysm node. Without an eth1 node block proposals for your " +
				"validator will be affected and the beacon node will not be able to initialize the genesis state",
		)
	}
	endpoints := []string{c.String(flags.HTTPWeb3ProviderFlag.Name)}
	endpoints = append(endpoints, c.StringSlice(flags.FallbackWeb3ProviderFlag.Name)...)
	return endpoints
}
