// This code was adapted from https://github.com/ethereum/go-ethereum/blob/master/cmd/geth/usage.go
package main

import (
	"io"
	"sort"
	"github.com/urfave/cli"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/debug"
)

var AppHelpTemplate = `NAME:
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
	Name string
	Flags []cli.Flag
}

var AppHelpFlagGroups = []flagGroup {
	{
		Name: "cmd",
			Flags: []cli.Flag {
				cmd.BootstrapNode,
				cmd.RelayNode,
				cmd.P2PPort,
				cmd.DataDirFlag,
				cmd.VerbosityFlag,
				cmd.EnableTracingFlag,
				cmd.TracingEndpointFlag,
				cmd.TraceSampleFractionFlag,
				cmd.MonitoringPortFlag,
				cmd.DisableMonitoringFlag,
			},
		},
	{
		Name: "debug",
		Flags: []cli.Flag {
			debug.PProfFlag,
			debug.PProfAddrFlag,
			debug.PProfPortFlag,
			debug.MemProfileRateFlag,
			debug.CPUProfileFlag,
			debug.TraceFlag,
		},
	},
	{
		Name: "utils",
		Flags: []cli.Flag {
			utils.DemoConfigFlag,
			utils.DepositContractFlag,
			utils.Web3ProviderFlag,
			utils.RPCPort,
			utils.CertFlag,
			utils.KeyFlag,
			utils.GenesisJSON,
			utils.EnablePOWChain,
			utils.EnableDBCleanup,
			utils.ChainStartDelay,
		},
	},
}

func init() {
	cli.AppHelpTemplate = AppHelpTemplate

	type helpData struct {
		App        interface{} 
		FlagGroups []flagGroup
	}

	originalHelpPrinter := cli.HelpPrinter
	cli.HelpPrinter = func(w io.Writer, tmpl string, data interface{}) {
		if (tmpl == AppHelpTemplate) {
			for _, group := range AppHelpFlagGroups {
				sort.Sort(cli.FlagsByName(group.Flags))
			}
			originalHelpPrinter(w, tmpl, helpData{data, AppHelpFlagGroups})
		} else {
			originalHelpPrinter(w, tmpl, data)
		}
	}
}
