package pandora

import "github.com/urfave/cli/v2"

var (
	// PandoraRPCProviderFlag defines a pandora node RPC endpoint
	PandoraRpcIpcProviderFlag = &cli.StringFlag{
		Name:  "pandora-ipc-provider",
		Usage: "Filename for IPC socket/pipe of pandora client.",
	}
	// PandoraRPCProviderFlag defines a pandora node RPC endpoint
	PandoraRpcHttpProviderFlag = &cli.StringFlag{
		Name:  "pandora-http-provider",
		Usage: "A pandora string rpc endpoint. This is our pandora client http endpoint.",
		Value: "http://127.0.0.1:8454",
	}
)
