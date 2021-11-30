package node

import (
	"github.com/ethereum/go-ethereum/common"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/cmd"
	"github.com/prysmaticlabs/prysm/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/config/params"
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

func configureChainConfig(cliCtx *cli.Context) {
	if cliCtx.IsSet(cmd.ChainConfigFileFlag.Name) {
		chainConfigFileName := cliCtx.String(cmd.ChainConfigFileFlag.Name)
		params.LoadChainConfigFile(chainConfigFileName)
	}
}

func configureHistoricalSlasher(cliCtx *cli.Context) {
	if cliCtx.Bool(flags.HistoricalSlasherNode.Name) {
		c := params.BeaconConfig()
		// Save a state every 4 epochs.
		c.SlotsPerArchivedPoint = params.BeaconConfig().SlotsPerEpoch * 4
		params.OverrideBeaconConfig(c)
		cmdConfig := cmd.Get()
		// Allow up to 4096 attestations at a time to be requested from the beacon nde.
		cmdConfig.MaxRPCPageSize = int(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().MaxAttestations))
		cmd.Init(cmdConfig)
		log.Warnf(
			"Setting %d slots per archive point and %d max RPC page size for historical slasher usage. This requires additional storage",
			c.SlotsPerArchivedPoint,
			cmdConfig.MaxRPCPageSize,
		)
	}
}

func configureSlotsPerArchivedPoint(cliCtx *cli.Context) {
	if cliCtx.IsSet(flags.SlotsPerArchivedPoint.Name) {
		c := params.BeaconConfig()
		c.SlotsPerArchivedPoint = types.Slot(cliCtx.Int(flags.SlotsPerArchivedPoint.Name))
		params.OverrideBeaconConfig(c)
	}
}

func configureEth1Config(cliCtx *cli.Context) {
	if cliCtx.IsSet(flags.ChainID.Name) {
		c := params.BeaconConfig()
		c.DepositChainID = cliCtx.Uint64(flags.ChainID.Name)
		params.OverrideBeaconConfig(c)
	}
	if cliCtx.IsSet(flags.NetworkID.Name) {
		c := params.BeaconConfig()
		c.DepositNetworkID = cliCtx.Uint64(flags.NetworkID.Name)
		params.OverrideBeaconConfig(c)
	}
	if cliCtx.IsSet(flags.DepositContractFlag.Name) {
		c := params.BeaconConfig()
		c.DepositContractAddress = cliCtx.String(flags.DepositContractFlag.Name)
		params.OverrideBeaconConfig(c)
	}
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

func configureInteropConfig(cliCtx *cli.Context) {
	genStateIsSet := cliCtx.IsSet(flags.InteropGenesisStateFlag.Name)
	genTimeIsSet := cliCtx.IsSet(flags.InteropGenesisTimeFlag.Name)
	numValsIsSet := cliCtx.IsSet(flags.InteropNumValidatorsFlag.Name)
	votesIsSet := cliCtx.IsSet(flags.InteropMockEth1DataVotesFlag.Name)

	if genStateIsSet || genTimeIsSet || numValsIsSet || votesIsSet {
		bCfg := params.BeaconConfig()
		bCfg.ConfigName = "interop"
		params.OverrideBeaconConfig(bCfg)
	}
}

func configureExecutionSetting(cliCtx *cli.Context) {
	if cliCtx.IsSet(flags.TerminalTotalDifficultyOverride.Name) {
		c := params.BeaconConfig()
		c.TerminalTotalDifficulty = cliCtx.Uint64(flags.TerminalTotalDifficultyOverride.Name)
		params.OverrideBeaconConfig(c)
	}
	if cliCtx.IsSet(flags.TerminalBlockHashOverride.Name) {
		c := params.BeaconConfig()
		c.TerminalBlockHash = common.HexToHash(cliCtx.String(flags.TerminalBlockHashOverride.Name))
		params.OverrideBeaconConfig(c)
	}
	if cliCtx.IsSet(flags.TerminalBlockHashActivationEpochOverride.Name) {
		c := params.BeaconConfig()
		c.TerminalBlockHashActivationEpoch = types.Epoch(cliCtx.Uint64(flags.TerminalBlockHashActivationEpochOverride.Name))
		params.OverrideBeaconConfig(c)
	}
	if cliCtx.IsSet(flags.FeeRecipient.Name) {
		c := params.BeaconConfig()
		c.FeeRecipient = common.HexToAddress(cliCtx.String(flags.FeeRecipient.Name))
		params.OverrideBeaconConfig(c)
	}
}
