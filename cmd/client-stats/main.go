package main

import (
	"fmt"
	"os"
	runtimeDebug "runtime/debug"
	"time"

	joonix "github.com/joonix/log"
	"github.com/prysmaticlabs/prysm/cmd"
	"github.com/prysmaticlabs/prysm/cmd/client-stats/flags"
	"github.com/prysmaticlabs/prysm/io/logs"
	"github.com/prysmaticlabs/prysm/monitoring/clientstats"
	"github.com/prysmaticlabs/prysm/monitoring/journald"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

var appFlags = []cli.Flag{
	cmd.VerbosityFlag,
	cmd.LogFormat,
	cmd.LogFileName,
	cmd.ConfigFileFlag,
	flags.BeaconnodeMetricsURLFlag,
	flags.ValidatorMetricsURLFlag,
	flags.ClientStatsAPIURLFlag,
	flags.ScrapeIntervalFlag,
}

func main() {
	app := cli.App{}
	app.Name = "client-stats"
	app.Usage = "daemon to scrape client-stats from prometheus and ship to a remote endpoint"
	app.Action = run
	app.Version = version.Version()

	app.Flags = appFlags

	// logging/config setup cargo-culted from beaconchain
	app.Before = func(ctx *cli.Context) error {
		// Load flags from config file, if specified.
		if err := cmd.LoadFlagsFromConfig(ctx, app.Flags); err != nil {
			return err
		}

		verbosity := ctx.String(cmd.VerbosityFlag.Name)
		level, err := logrus.ParseLevel(verbosity)
		if err != nil {
			return err
		}
		logrus.SetLevel(level)

		format := ctx.String(cmd.LogFormat.Name)
		switch format {
		case "text":
			formatter := new(prefixed.TextFormatter)
			formatter.TimestampFormat = "2006-01-02 15:04:05"
			formatter.FullTimestamp = true
			// If persistent log files are written - we disable the log messages coloring because
			// the colors are ANSI codes and seen as gibberish in the log files.
			formatter.DisableColors = ctx.String(cmd.LogFileName.Name) != ""
			logrus.SetFormatter(formatter)
		case "fluentd":
			f := joonix.NewFormatter()
			if err := joonix.DisableTimestampFormat(f); err != nil {
				panic(err)
			}
			logrus.SetFormatter(f)
		case "json":
			logrus.SetFormatter(&logrus.JSONFormatter{})
		case "journald":
			if err := journald.Enable(); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown log format %s", format)
		}

		logFileName := ctx.String(cmd.LogFileName.Name)
		if logFileName != "" {
			if err := logs.ConfigurePersistentLogging(logFileName); err != nil {
				log.WithError(err).Error("Failed to configuring logging to disk.")
			}
		}
		return cmd.ValidateNoArgs(ctx)
	}

	defer func() {
		if x := recover(); x != nil {
			log.Errorf("Runtime panic: %v\n%v", x, string(runtimeDebug.Stack()))
			panic(x)
		}
	}()

	if err := app.Run(os.Args); err != nil {
		log.Error(err.Error())
	}
}

func run(ctx *cli.Context) error {
	var upd clientstats.Updater
	if ctx.IsSet(flags.ClientStatsAPIURLFlag.Name) {
		u := ctx.String(flags.ClientStatsAPIURLFlag.Name)
		upd = clientstats.NewClientStatsHTTPPostUpdater(u)
	} else {
		log.Warn("No --clientstats-api-url flag set, writing to stdout as default metrics sink.")
		upd = clientstats.NewGenericClientStatsUpdater(os.Stdout)
	}

	scrapers := make([]clientstats.Scraper, 0)
	if ctx.IsSet(flags.BeaconnodeMetricsURLFlag.Name) {
		u := ctx.String(flags.BeaconnodeMetricsURLFlag.Name)
		scrapers = append(scrapers, clientstats.NewBeaconNodeScraper(u))
	}
	if ctx.IsSet(flags.ValidatorMetricsURLFlag.Name) {
		u := ctx.String(flags.ValidatorMetricsURLFlag.Name)
		scrapers = append(scrapers, clientstats.NewValidatorScraper(u))
	}

	ticker := time.NewTicker(ctx.Duration(flags.ScrapeIntervalFlag.Name))
	for {
		select {
		case <-ticker.C:
			for _, s := range scrapers {
				r, err := s.Scrape()
				if err != nil {
					log.Errorf("Scraper error: %s", err)
					continue
				}
				err = upd.Update(r)
				if err != nil {
					log.Errorf("client-stats collector error: %s", err)
					continue
				}
			}
		case <-ctx.Done():
			ticker.Stop()
			return nil
		}
	}
}
