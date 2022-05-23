package node

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/cmd"
	"github.com/prysmaticlabs/prysm/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/config/params"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	tracing2 "github.com/prysmaticlabs/prysm/monitoring/tracing"
	"github.com/urfave/cli/v2"
)

func configureTracing(cliCtx *cli.Context) error {
	return tracing2.Setup(
		"beacon-chain", // service name
		cliCtx.String(cmd.TracingProcessNameFlag.Name),
		cliCtx.String(cmd.TracingEndpointFlag.Name),
		cliCtx.Float64(cmd.TraceSampleFractionFlag.Name),
		cliCtx.Bool(cmd.EnableTracingFlag.Name),
	)
}

func configureChainConfig(cliCtx *cli.Context) error {
	if cliCtx.IsSet(cmd.ChainConfigFileFlag.Name) {
		chainConfigFileName := cliCtx.String(cmd.ChainConfigFileFlag.Name)
		return params.LoadChainConfigFile(chainConfigFileName, nil)
	}
	return nil
}

func configureHistoricalSlasher(cliCtx *cli.Context) error {
	if cliCtx.Bool(flags.HistoricalSlasherNode.Name) {
		c := params.BeaconConfig().Copy()
		// Save a state every 4 epochs.
		c.SlotsPerArchivedPoint = params.BeaconConfig().SlotsPerEpoch * 4
		if err := params.SetActive(c); err != nil {
			return err
		}
		cmdConfig := cmd.Get()
		// Allow up to 4096 attestations at a time to be requested from the beacon nde.
		cmdConfig.MaxRPCPageSize = int(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().MaxAttestations)) // lint:ignore uintcast -- Page size should not exceed int64 with these constants.
		cmd.Init(cmdConfig)
		log.Warnf(
			"Setting %d slots per archive point and %d max RPC page size for historical slasher usage. This requires additional storage",
			c.SlotsPerArchivedPoint,
			cmdConfig.MaxRPCPageSize,
		)
	}
	return nil
}

func configureSafeSlotsToImportOptimistically(cliCtx *cli.Context) error {
	if cliCtx.IsSet(flags.SafeSlotsToImportOptimistically.Name) {
		c := params.BeaconConfig().Copy()
		c.SafeSlotsToImportOptimistically = types.Slot(cliCtx.Int(flags.SafeSlotsToImportOptimistically.Name))
		if err := params.SetActive(c); err != nil {
			return err
		}
	}
	return nil
}

func configureSlotsPerArchivedPoint(cliCtx *cli.Context) error {
	if cliCtx.IsSet(flags.SlotsPerArchivedPoint.Name) {
		c := params.BeaconConfig().Copy()
		c.SlotsPerArchivedPoint = types.Slot(cliCtx.Int(flags.SlotsPerArchivedPoint.Name))
		if err := params.SetActive(c); err != nil {
			return err
		}
	}
	return nil
}

func configureEth1Config(cliCtx *cli.Context) error {
	c := params.BeaconConfig().Copy()
	if cliCtx.IsSet(flags.ChainID.Name) {
		c.DepositChainID = cliCtx.Uint64(flags.ChainID.Name)
		if err := params.SetActive(c); err != nil {
			return err
		}
	}
	if cliCtx.IsSet(flags.NetworkID.Name) {
		c.DepositNetworkID = cliCtx.Uint64(flags.NetworkID.Name)
		if err := params.SetActive(c); err != nil {
			return err
		}
	}
	if cliCtx.IsSet(flags.DepositContractFlag.Name) {
		c.DepositContractAddress = cliCtx.String(flags.DepositContractFlag.Name)
		if err := params.SetActive(c); err != nil {
			return err
		}
	}
	return nil
}

func configureNetwork(cliCtx *cli.Context) {
	if cliCtx.IsSet(cmd.BootstrapNode.Name) {
		c := params.BeaconNetworkConfig()
		c.BootstrapNodes = cliCtx.StringSlice(cmd.BootstrapNode.Name)
		params.OverrideBeaconNetworkConfig(c)
	}
	if cliCtx.IsSet(flags.ContractDeploymentBlock.Name) {
		networkCfg := params.BeaconNetworkConfig()
		networkCfg.ContractDeploymentBlock = uint64(cliCtx.Int(flags.ContractDeploymentBlock.Name))
		params.OverrideBeaconNetworkConfig(networkCfg)
	}
}

func configureInteropConfig(cliCtx *cli.Context) error {
	// an explicit chain config was specified, don't mess with it
	if cliCtx.IsSet(cmd.ChainConfigFileFlag.Name) {
		return nil
	}
	genStateIsSet := cliCtx.IsSet(flags.InteropGenesisStateFlag.Name)
	genTimeIsSet := cliCtx.IsSet(flags.InteropGenesisTimeFlag.Name)
	numValsIsSet := cliCtx.IsSet(flags.InteropNumValidatorsFlag.Name)
	votesIsSet := cliCtx.IsSet(flags.InteropMockEth1DataVotesFlag.Name)

	if genStateIsSet || genTimeIsSet || numValsIsSet || votesIsSet {
		if err := params.SetActive(params.InteropConfig().Copy()); err != nil {
			return err
		}
	}
	return nil
}

func configureExecutionSetting(cliCtx *cli.Context) error {
	if !cliCtx.IsSet(flags.SuggestedFeeRecipient.Name) {
		return nil
	}

	c := params.BeaconConfig().Copy()
	ha := cliCtx.String(flags.SuggestedFeeRecipient.Name)
	if !common.IsHexAddress(ha) {
		return fmt.Errorf("%s is not a valid fee recipient address", ha)
	}
	mixedcaseAddress, err := common.NewMixedcaseAddressFromString(ha)
	if err != nil {
		return errors.Wrapf(err, "could not decode fee recipient %s", ha)
	}
	checksumAddress := common.HexToAddress(ha)
	if !mixedcaseAddress.ValidChecksum() {
		log.Warnf("Fee recipient %s is not a checksum Ethereum address. "+
			"The checksummed address is %s and will be used as the fee recipient. "+
			"We recommend using a mixed-case address (checksum) "+
			"to prevent spelling mistakes in your fee recipient Ethereum address", ha, checksumAddress.Hex())
	}
	c.DefaultFeeRecipient = checksumAddress
	return params.SetActive(c)
}
