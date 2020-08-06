// This code was adapted from https://github.com/ethereum/go-ethereum/blob/master/cmd/geth/usage.go
package main

import (
	"io"
	"sort"

	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/debug"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/urfave/cli/v2"
)

var appHelpTemplate = `NAME:
   {{.App.Name}} - {{.App.Usage}}
USAGE:
   {{.App.HelpName}} [options]{{if .App.Commands}} command [command options]{{end}} {{if .App.ArgsUsage}}{{.App.ArgsUsage}}{{else}}[arguments...]{{end}}
   {{if .App.Version}}
AUTHOR:
   {{range .App.Authors}}{{ . }}{{end}}
   {{end}}{{if .App.Commands}}
GLOBAL OPTIONS:
   {{range .App.Commands}}{{join .Names ", "}}{{ "\t" }}{{.Usage}}
   {{end}}{{end}}{{if .FlagGroups}}
{{range .FlagGroups}}{{.Name}} OPTIONS:
   {{range .Flags}}{{.}}
   {{end}}
{{end}}{{end}}{{if .App.Copyright }}
COPYRIGHT:
   {{.App.Copyright}}
VERSION:
   {{.App.Version}}
   {{end}}{{if len .App.Authors}}
   {{end}}
`

type flagGroup struct {
	Name  string
	Flags []cli.Flag
}

var appHelpFlagGroups = []flagGroup{
	{
		Name: "cmd",
		Flags: []cli.Flag{
			cmd.MinimalConfigFlag,
			cmd.E2EConfigFlag,
			cmd.RPCMaxPageSizeFlag,
			cmd.NoDiscovery,
			cmd.BootstrapNode,
			cmd.RelayNode,
			cmd.P2PUDPPort,
			cmd.P2PTCPPort,
			cmd.DataDirFlag,
			cmd.VerbosityFlag,
			cmd.EnableTracingFlag,
			cmd.TracingProcessNameFlag,
			cmd.TracingEndpointFlag,
			cmd.TraceSampleFractionFlag,
			cmd.MonitoringHostFlag,
			flags.MonitoringPortFlag,
			cmd.DisableMonitoringFlag,
			cmd.MaxGoroutines,
			cmd.ForceClearDB,
			cmd.ClearDB,
			cmd.ConfigFileFlag,
			cmd.ChainConfigFileFlag,
			cmd.GrpcMaxCallRecvMsgSizeFlag,
		},
	},
	{
		Name: "debug",
		Flags: []cli.Flag{
			debug.PProfFlag,
			debug.PProfAddrFlag,
			debug.PProfPortFlag,
			debug.MemProfileRateFlag,
			debug.CPUProfileFlag,
			debug.TraceFlag,
		},
	},
	{
		Name: "beacon-chain",
		Flags: []cli.Flag{
			flags.InteropMockEth1DataVotesFlag,
			flags.InteropGenesisStateFlag,
			flags.DepositContractFlag,
			flags.ContractDeploymentBlock,
			flags.RPCHost,
			flags.RPCPort,
			flags.CertFlag,
			flags.KeyFlag,
			flags.DisableGRPCGateway,
			flags.GRPCGatewayHost,
			flags.GRPCGatewayPort,
			flags.HTTPWeb3ProviderFlag,
			flags.SetGCPercent,
			flags.UnsafeSync,
			flags.SlasherCertFlag,
			flags.SlasherProviderFlag,
			flags.SlotsPerArchivedPoint,
			flags.DisableDiscv5,
			flags.BlockBatchLimit,
			flags.BlockBatchLimitBurstFactor,
			flags.EnableDebugRPCEndpoints,
			flags.SlotsPerArchivedPoint,
			flags.HistoricalSlasherNode,
			flags.ChainID,
			flags.NetworkID,
			flags.EnableRoughtime,
		},
	},
	{
		Name: "p2p",
		Flags: []cli.Flag{
			cmd.P2PIP,
			cmd.P2PHost,
			cmd.P2PHostDNS,
			cmd.P2PMaxPeers,
			cmd.P2PPrivKey,
			cmd.P2PMetadata,
			cmd.P2PAllowList,
			cmd.P2PDenyList,
			cmd.StaticPeers,
			cmd.EnableUPnPFlag,
			flags.MinSyncPeers,
		},
	},
	{
		Name: "log",
		Flags: []cli.Flag{
			cmd.LogFormat,
			cmd.LogFileName,
		},
	},
	{
		Name:  "features",
		Flags: featureconfig.ActiveFlags(featureconfig.BeaconChainFlags),
	},
	{
		Name: "interop",
		Flags: []cli.Flag{
			flags.InteropGenesisStateFlag,
			flags.InteropGenesisTimeFlag,
			flags.InteropNumValidatorsFlag,
		},
	},
}

func init() {
	cli.AppHelpTemplate = appHelpTemplate

	type helpData struct {
		App        interface{}
		FlagGroups []flagGroup
	}

	originalHelpPrinter := cli.HelpPrinter
	cli.HelpPrinter = func(w io.Writer, tmpl string, data interface{}) {
		if tmpl == appHelpTemplate {
			for _, group := range appHelpFlagGroups {
				sort.Sort(cli.FlagsByName(group.Flags))
			}
			originalHelpPrinter(w, tmpl, helpData{data, appHelpFlagGroups})
		} else {
			originalHelpPrinter(w, tmpl, data)
		}
	}
}
