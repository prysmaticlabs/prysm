// This code was adapted from https://github.com/ethereum/go-ethereum/blob/master/cmd/geth/usage.go
package main

import (
	"io"
	"sort"

	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/debug"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/urfave/cli"
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
			cmd.VerbosityFlag,
			cmd.DataDirFlag,
			cmd.EnableTracingFlag,
			cmd.TracingProcessNameFlag,
			cmd.TracingEndpointFlag,
			cmd.TraceSampleFractionFlag,
			cmd.BootstrapNode,
			cmd.MonitoringPortFlag,
			cmd.LogFormat,
			cmd.LogFileName,
			cmd.EnableUPnPFlag,
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
		Name: "validator",
		Flags: []cli.Flag{
			flags.NoCustomConfigFlag,
			flags.BeaconRPCProviderFlag,
			flags.CertFlag,
			flags.KeystorePathFlag,
			flags.PasswordFlag,
			flags.DisablePenaltyRewardLogFlag,
			flags.UnencryptedKeysFlag,
		},
	},
	{
		Name:  "features",
		Flags: featureconfig.ValidatorFlags,
	},
	{
		Name: "interop",
		Flags: []cli.Flag{
			flags.InteropNumValidators,
			flags.InteropStartIndex,
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
