package slashersimulator

import (
	"runtime"

	"github.com/prysmaticlabs/prysm/shared/logutil"
	log "github.com/sirupsen/logrus"

	"github.com/prysmaticlabs/prysm/beacon-chain/slasher/simulator"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/debug"
	"github.com/urfave/cli/v2"
)

// Commands for slasher simulation
var Commands = &cli.Command{
	Name:     "slasher-simulator",
	Category: "slasher",
	Usage:    "defines commands for simulating a slasher acting at real-scale",
	Flags: cmd.WrapFlags([]cli.Flag{
		cmd.DataDirFlag,
		cmd.ClearDB,
		cmd.ForceClearDB,
		cmd.ConfigFileFlag,
		cmd.LogFileName,
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

		logFileName := cliCtx.String(cmd.LogFileName.Name)
		if logFileName != "" {
			if err := logutil.ConfigurePersistentLogging(logFileName); err != nil {
				log.WithError(err).Error("Failed to configuring logging to disk.")
			}
		}
		return debug.Setup(cliCtx)
	},
	After: func(ctx *cli.Context) error {
		debug.Exit(ctx)
		return nil
	},
	Action: simulator.Simulate,
}
