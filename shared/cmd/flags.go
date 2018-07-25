package cmd

import (
	"github.com/ethereum/go-ethereum/node"
	"github.com/urfave/cli"
)

var (
	// VerbosityFlag defines the logrus configuration.
	VerbosityFlag = cli.StringFlag{
		Name:  "verbosity",
		Usage: "Logging verbosity (debug, info=default, warn, error, fatal, panic)",
		Value: "info",
	}
	// IPCPathFlag defines the filename of a pipe within the datadir.
	IPCPathFlag = DirectoryFlag{
		Name:  "ipcpath",
		Usage: "Filename for IPC socket/pipe within the datadir (explicit paths escape it)",
	}
	// DataDirFlag defines a path on disk.
	DataDirFlag = DirectoryFlag{
		Name:  "datadir",
		Usage: "Data directory for the databases and keystore",
		Value: DirectoryString{node.DefaultDataDir()},
	}
	// NetworkIdFlag defines the specific network identifier.
	NetworkIdFlag = cli.Uint64Flag{
		Name:  "networkid",
		Usage: "Network identifier (integer, 1=Frontier, 2=Morden (disused), 3=Ropsten, 4=Rinkeby)",
		Value: 1,
	}
	// PasswordFileFlag defines the path to the user's account password file.
	PasswordFileFlag = cli.StringFlag{
		Name:  "password",
		Usage: "Password file to use for non-interactive password input",
		Value: "",
	}
	// RPCProviderFlag defines a http endpoint flag to connect to mainchain.
	RPCProviderFlag = cli.StringFlag{
		Name:  "rpc",
		Usage: "HTTP-RPC server end point to use to connect to mainchain.",
		Value: "http://localhost:8545/",
	}
)
