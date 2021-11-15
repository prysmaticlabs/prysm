// Package cmd defines the command line flags for the shared utlities.
package cmd

import (
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

var (
	// MinimalConfigFlag declares to use the minimal config for running Ethereum consensus.
	MinimalConfigFlag = &cli.BoolFlag{
		Name:  "minimal-config",
		Usage: "Use minimal config with parameters as defined in the spec.",
	}
	// E2EConfigFlag declares to use a testing specific config for running Ethereum consensus in end-to-end testing.
	E2EConfigFlag = &cli.BoolFlag{
		Name:  "e2e-config",
		Usage: "Use the E2E testing config, only for use within end-to-end testing.",
	}
	// RPCMaxPageSizeFlag defines the maximum numbers per page returned in RPC responses from this
	// beacon node (default: 500).
	RPCMaxPageSizeFlag = &cli.IntFlag{
		Name:  "rpc-max-page-size",
		Usage: "Max number of items returned per page in RPC responses for paginated endpoints.",
	}
	// VerbosityFlag defines the logrus configuration.
	VerbosityFlag = &cli.StringFlag{
		Name:  "verbosity",
		Usage: "Logging verbosity (trace, debug, info=default, warn, error, fatal, panic)",
		Value: "info",
	}
	// DataDirFlag defines a path on disk where Prysm databases are stored.
	DataDirFlag = &cli.StringFlag{
		Name:  "datadir",
		Usage: "Data directory for the databases",
		Value: DefaultDataDir(),
	}
	// EnableBackupWebhookFlag for users to trigger db backups via an HTTP webhook.
	EnableBackupWebhookFlag = &cli.BoolFlag{
		Name:  "enable-db-backup-webhook",
		Usage: "Serve HTTP handler to initiate database backups. The handler is served on the monitoring port at path /db/backup.",
	}
	// BackupWebhookOutputDir to customize the output directory for db backups.
	BackupWebhookOutputDir = &cli.StringFlag{
		Name:  "db-backup-output-dir",
		Usage: "Output directory for db backups",
	}
	// EnableTracingFlag defines a flag to enable p2p message tracing.
	EnableTracingFlag = &cli.BoolFlag{
		Name:  "enable-tracing",
		Usage: "Enable request tracing.",
	}
	// TracingProcessNameFlag defines a flag to specify a process name.
	TracingProcessNameFlag = &cli.StringFlag{
		Name:  "tracing-process-name",
		Usage: "The name to apply to tracing tag \"process_name\"",
	}
	// TracingEndpointFlag flag defines the http endpoint for serving traces to Jaeger.
	TracingEndpointFlag = &cli.StringFlag{
		Name:  "tracing-endpoint",
		Usage: "Tracing endpoint defines where beacon chain traces are exposed to Jaeger.",
		Value: "http://127.0.0.1:14268/api/traces",
	}
	// TraceSampleFractionFlag defines a flag to indicate what fraction of p2p
	// messages are sampled for tracing.
	TraceSampleFractionFlag = &cli.Float64Flag{
		Name:  "trace-sample-fraction",
		Usage: "Indicate what fraction of p2p messages are sampled for tracing.",
		Value: 0.20,
	}
	// MonitoringHostFlag defines the host used to serve prometheus metrics.
	MonitoringHostFlag = &cli.StringFlag{
		Name:  "monitoring-host",
		Usage: "Host used for listening and responding metrics for prometheus.",
		Value: "127.0.0.1",
	}
	// DisableMonitoringFlag defines a flag to disable the metrics collection.
	DisableMonitoringFlag = &cli.BoolFlag{
		Name:  "disable-monitoring",
		Usage: "Disable monitoring service.",
	}
	// ForceClearDB removes any previously stored data at the data directory.
	ForceClearDB = &cli.BoolFlag{
		Name:  "force-clear-db",
		Usage: "Clear any previously stored data at the data directory",
	}
	// ClearDB prompts user to see if they want to remove any previously stored data at the data directory.
	ClearDB = &cli.BoolFlag{
		Name:  "clear-db",
		Usage: "Prompt for clearing any previously stored data at the data directory",
	}
	// LogFormat specifies the log output format.
	LogFormat = &cli.StringFlag{
		Name:  "log-format",
		Usage: "Specify log formatting. Supports: text, json, fluentd, journald.",
		Value: "text",
	}
	// MaxGoroutines specifies the maximum amount of goroutines tolerated, before a status check fails.
	MaxGoroutines = &cli.IntFlag{
		Name:  "max-goroutines",
		Usage: "Specifies the upper limit of goroutines running before a status check fails",
		Value: 5000,
	}
	// LogFileName specifies the log output file name.
	LogFileName = &cli.StringFlag{
		Name:  "log-file",
		Usage: "Specify log file name, relative or absolute",
	}
	// ConfigFileFlag specifies the filepath to load flag values.
	ConfigFileFlag = &cli.StringFlag{
		Name:  "config-file",
		Usage: "The filepath to a yaml file with flag values",
	}
	// ChainConfigFileFlag specifies the filepath to load flag values.
	ChainConfigFileFlag = &cli.StringFlag{
		Name:  "chain-config-file",
		Usage: "The path to a YAML file with chain config values",
	}
	// GrpcMaxCallRecvMsgSizeFlag defines the max call message size for GRPC
	GrpcMaxCallRecvMsgSizeFlag = &cli.IntFlag{
		Name:  "grpc-max-msg-size",
		Usage: "Integer to define max recieve message call size (default: 4194304 (for 4MB))",
		Value: 1 << 22,
	}
	// AcceptTosFlag specifies user acceptance of ToS for non-interactive environments.
	AcceptTosFlag = &cli.BoolFlag{
		Name:  "accept-terms-of-use",
		Usage: "Accept Terms and Conditions (for non-interactive environments)",
	}
	// RestoreSourceFileFlag specifies the filepath to the backed-up database file
	// which will be used to restore the database.
	RestoreSourceFileFlag = &cli.StringFlag{
		Name:  "restore-source-file",
		Usage: "Filepath to the backed-up database file which will be used to restore the database",
	}
	// RestoreTargetDirFlag specifies the target directory of the restored database.
	RestoreTargetDirFlag = &cli.StringFlag{
		Name:  "restore-target-dir",
		Usage: "Target directory of the restored database",
		Value: DefaultDataDir(),
	}
	// BoltMMapInitialSizeFlag specifies the initial size in bytes of boltdb's mmap syscall.
	BoltMMapInitialSizeFlag = &cli.IntFlag{
		Name:  "bolt-mmap-initial-size",
		Usage: "Specifies the size in bytes of bolt db's mmap syscall allocation",
		Value: 536870912, // 512 Mb as a default value.
	}
)

// LoadFlagsFromConfig sets flags values from config file if ConfigFileFlag is set.
func LoadFlagsFromConfig(cliCtx *cli.Context, flags []cli.Flag) error {
	if cliCtx.IsSet(ConfigFileFlag.Name) {
		if err := altsrc.InitInputSourceWithContext(flags, altsrc.NewYamlSourceFromFlagFunc(ConfigFileFlag.Name))(cliCtx); err != nil {
			return err
		}
	}
	return nil
}

// ValidateNoArgs insures that the application is not run with erroneous arguments or flags.
// This function should be used in the app.Before, whenever the application supports a default command.
func ValidateNoArgs(ctx *cli.Context) error {
	commandList := ctx.App.Commands
	parentCommand := ctx.Command
	isParamForFlag := false
	for _, a := range ctx.Args().Slice() {
		// We don't validate further if
		// the following value is actually
		// a parameter for a flag.
		if isParamForFlag {
			isParamForFlag = false
			continue
		}
		if strings.HasPrefix(a, "-") || strings.HasPrefix(a, "--") {
			// In the event our flag doesn't specify
			// the relevant argument with an equal
			// sign, we can assume the next argument
			// is the relevant value for the flag.
			flagName := strings.TrimPrefix(a, "--")
			flagName = strings.TrimPrefix(flagName, "-")
			if !strings.Contains(a, "=") && !isBoolFlag(parentCommand, flagName) {
				isParamForFlag = true
			}
			continue
		}
		c := checkCommandList(commandList, a)
		if c == nil {
			return fmt.Errorf("unrecognized argument: %s", a)
		}
		// Set the command list as the subcommand's
		// from the current selected parent command.
		commandList = c.Subcommands
		parentCommand = c
	}
	return nil
}

// verifies that the provided command is in the command list.
func checkCommandList(commands []*cli.Command, name string) *cli.Command {
	for _, c := range commands {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func isBoolFlag(com *cli.Command, name string) bool {
	for _, f := range com.Flags {
		switch bFlag := f.(type) {
		case *cli.BoolFlag:
			if bFlag.Name == name {
				return true
			}
		case *altsrc.BoolFlag:
			if bFlag.Name == name {
				return true
			}
		}
	}
	return false
}
