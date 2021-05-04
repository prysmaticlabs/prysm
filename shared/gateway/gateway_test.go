package gateway

import (
	"flag"
	"fmt"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"github.com/urfave/cli/v2"
)

// Test that beacon gateway Start, Stop.
func TestBeaconGateway_StartStop(t *testing.T) {
	hook := logTest.NewGlobal()

	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	ctx := cli.NewContext(&app, set, nil)

	gatewayPort := ctx.Int(flags.GRPCGatewayPort.Name)
	gatewayHost := ctx.String(flags.GRPCGatewayHost.Name)
	rpcHost := ctx.String(flags.RPCHost.Name)
	selfAddress := fmt.Sprintf("%s:%d", rpcHost, ctx.Int(flags.RPCPort.Name))
	gatewayAddress := fmt.Sprintf("%s:%d", gatewayHost, gatewayPort)
	allowedOrigins := strings.Split(ctx.String(flags.GPRCGatewayCorsDomain.Name), ",")
	enableDebugRPCEndpoints := ctx.Bool(flags.EnableDebugRPCEndpoints.Name)
	selfCert := ctx.String(flags.CertFlag.Name)

	beaconGateway := NewBeacon(
		ctx.Context,
		selfAddress,
		selfCert,
		gatewayAddress,
		nil, /*optional mux*/
		allowedOrigins,
		enableDebugRPCEndpoints,
		ctx.Uint64("grpc-max-msg-size"),
	)

	beaconGateway.Start()
	go func() {
		require.LogsContain(t, hook, "Starting gRPC gateway")
	}()

	err := beaconGateway.Stop()
	require.NoError(t, err)

}

// Test that validator gateway Start, Stop.
func TestValidatorGateway_StartStop(t *testing.T) {
	hook := logTest.NewGlobal()

	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	ctx := cli.NewContext(&app, set, nil)

	gatewayHost := ctx.String(flags.GRPCGatewayHost.Name)
	gatewayPort := ctx.Int(flags.GRPCGatewayPort.Name)
	rpcHost := ctx.String(flags.RPCHost.Name)
	rpcPort := ctx.Int(flags.RPCPort.Name)
	rpcAddr := fmt.Sprintf("%s:%d", rpcHost, rpcPort)
	gatewayAddress := fmt.Sprintf("%s:%d", gatewayHost, gatewayPort)
	allowedOrigins := strings.Split(ctx.String(flags.GPRCGatewayCorsDomain.Name), ",")

	validatorGateway := NewValidator(
		ctx.Context,
		rpcAddr,
		gatewayAddress,
		allowedOrigins,
	)

	validatorGateway.Start()
	go func() {
		require.LogsContain(t, hook, "Starting gRPC gateway")
	}()

	err := validatorGateway.Stop()
	require.NoError(t, err)

}
