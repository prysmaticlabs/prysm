package gateway

import (
	"flag"
	"fmt"
	"net/http"
	"testing"

	"github.com/prysmaticlabs/prysm/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"github.com/urfave/cli/v2"
)

func TestGateway_StartStop(t *testing.T) {
	hook := logTest.NewGlobal()

	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	ctx := cli.NewContext(&app, set, nil)

	gatewayPort := ctx.Int(flags.GRPCGatewayPort.Name)
	gatewayHost := ctx.String(flags.GRPCGatewayHost.Name)
	rpcHost := ctx.String(flags.RPCHost.Name)
	selfAddress := fmt.Sprintf("%s:%d", rpcHost, ctx.Int(flags.RPCPort.Name))
	gatewayAddress := fmt.Sprintf("%s:%d", gatewayHost, gatewayPort)

	g := New(
		ctx.Context,
		[]PbHandlerRegistration{},
		func(handler http.Handler, writer http.ResponseWriter, request *http.Request) {

		},
		selfAddress,
		gatewayAddress,
	)

	g.Start()
	go func() {
		require.LogsContain(t, hook, "Starting gRPC gateway")
	}()

	err := g.Stop()
	require.NoError(t, err)
}
