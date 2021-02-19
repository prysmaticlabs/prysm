package simulator

import (
	"runtime"

	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/debug"
	"github.com/urfave/cli/v2"
)

var SlasherCommands = &cli.Command{
	Name:     "slasher-simulator",
	Category: "slasher",
	Usage:    "defines commands for simulating a slasher acting at real-scale",
	Flags: cmd.WrapFlags([]cli.Flag{
		cmd.DataDirFlag,
		debug.PProfFlag,
		debug.MemProfileRateFlag,
		debug.MutexProfileFractionFlag,
		debug.TraceFlag,
		debug.CPUProfileFlag,
		debug.PProfAddrFlag,
		debug.PProfPortFlag,
	}),
	Before: func(cliCtx *cli.Context) error {
		runtime.GOMAXPROCS(runtime.NumCPU())
		return debug.Setup(cliCtx)
	},
	After: func(ctx *cli.Context) error {
		debug.Exit(ctx)
		return nil
	},
	Action: func(cliCtx *cli.Context) error {
		return Simulate(cliCtx)
	},
}
