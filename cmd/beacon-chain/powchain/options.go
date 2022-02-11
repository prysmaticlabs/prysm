package powchaincmd

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/io/file"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var log = logrus.WithField("prefix", "cmd-powchain")

// FlagOptions for powchain service flag configurations.
func FlagOptions(c *cli.Context) ([]powchain.Option, error) {
	endpoints := parsePowchainEndpoints(c)
	executionEndpoint := parseExecutionEndpoint(c)
	jwtSecret, err := parseJWTSecret(c)
	if err != nil {
		return nil, errors.Wrap(err, "could not read JWT secret file for authenticating execution API")
	}
	opts := []powchain.Option{
		powchain.WithHttpEndpoints(endpoints),
		powchain.WithEth1HeaderRequestLimit(c.Uint64(flags.Eth1HeaderReqLimit.Name)),
	}
	if executionEndpoint != "" {
		opts = append(opts, powchain.WithExecutionEndpoint(executionEndpoint))
	}
	if len(jwtSecret) > 0 {
		opts = append(opts, powchain.WithExecutionClientJWTSecret(jwtSecret))
	}
	return opts, nil
}

func parseJWTSecret(c *cli.Context) ([]byte, error) {
	jwtSecretFile := c.String(flags.ExecutionJWTSecretFlag.Name)
	if jwtSecretFile == "" {
		return nil, nil
	}
	enc, err := file.ReadFileAsBytes(jwtSecretFile)
	if err != nil {
		return nil, err
	}
	return hexutil.Decode(string(enc))
}

func parsePowchainEndpoints(c *cli.Context) []string {
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

func parseExecutionEndpoint(c *cli.Context) string {
	return c.String(flags.ExecutionProviderFlag.Name)
}
