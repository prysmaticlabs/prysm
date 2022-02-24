package powchaincmd

import (
	"fmt"
	"strings"

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
	jwtSecret, err := parseJWTSecretFromFile(c)
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

// Parses a JWT secret from a file path. This secret is required when connecting to execution nodes
// over HTTP, and must be the same one used in Prysm and the execution node server Prysm is connecting to.
// The engine API specification here https://github.com/ethereum/execution-apis/blob/main/src/engine/authentication.md
// Explains how we should validate this secret and the format of the file a user can specify.
//
// The secret must be stored as a hex-encoded string within a file in the filesystem.
// If the --jwt-secret flag is provided to Prysm, but the file cannot be read, or does not contain a hex-encoded
// key of at least 256 bits, the client should treat this as an error and abort the startup.
func parseJWTSecretFromFile(c *cli.Context) ([]byte, error) {
	jwtSecretFile := c.String(flags.ExecutionJWTSecretFlag.Name)
	if jwtSecretFile == "" {
		return nil, nil
	}
	enc, err := file.ReadFileAsBytes(jwtSecretFile)
	if err != nil {
		return nil, err
	}
	strData := strings.TrimSpace(string(enc))
	if len(strData) == 0 {
		return nil, fmt.Errorf("provided JWT secret in file %s cannot be empty", jwtSecretFile)
	}
	secret, err := hexutil.Decode(strData)
	if err != nil {
		return nil, err
	}
	if len(secret) < 32 {
		return nil, errors.New("provided JWT secret should be a hex string of at least 32 bytes")
	}
	return secret, nil
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
