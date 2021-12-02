package gateway

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gorilla/mux"
	"github.com/prysmaticlabs/prysm/api/gateway/apimiddleware"
	"github.com/prysmaticlabs/prysm/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"github.com/urfave/cli/v2"
)

type mockEndpointFactory struct {
}

func (*mockEndpointFactory) Paths() []string {
	return []string{}
}

func (*mockEndpointFactory) Create(_ string) (*apimiddleware.Endpoint, error) {
	return nil, nil
}

func (*mockEndpointFactory) IsNil() bool {
	return false
}

func TestGateway_Customized(t *testing.T) {
	r := mux.NewRouter()
	cert := "cert"
	origins := []string{"origin"}
	size := uint64(100)
	endpointFactory := &mockEndpointFactory{}

	g := New(
		context.Background(),
		[]*PbMux{},
		func(
			_ *apimiddleware.ApiProxyMiddleware,
			_ http.HandlerFunc,
			_ http.ResponseWriter,
			_ *http.Request,
		) {
		},
		"",
		"",
	).WithRouter(r).
		WithRemoteCert(cert).
		WithAllowedOrigins(origins).
		WithMaxCallRecvMsgSize(size).
		WithApiMiddleware(endpointFactory)

	assert.Equal(t, r, g.router)
	assert.Equal(t, cert, g.remoteCert)
	require.Equal(t, 1, len(g.allowedOrigins))
	assert.Equal(t, origins[0], g.allowedOrigins[0])
	assert.Equal(t, size, g.maxCallRecvMsgSize)
	assert.Equal(t, endpointFactory, g.apiMiddlewareEndpointFactory)
}

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
		[]*PbMux{},
		func(
			_ *apimiddleware.ApiProxyMiddleware,
			_ http.HandlerFunc,
			_ http.ResponseWriter,
			_ *http.Request,
		) {
		},
		selfAddress,
		gatewayAddress,
	)

	g.Start()
	go func() {
		require.LogsContain(t, hook, "Starting gRPC gateway")
		require.LogsDoNotContain(t, hook, "Starting API middleware")
	}()

	err := g.Stop()
	require.NoError(t, err)
}

func TestGateway_NilHandler_NotFoundHandlerRegistered(t *testing.T) {
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
		[]*PbMux{},
		/* muxHandler */ nil,
		selfAddress,
		gatewayAddress,
	)

	writer := httptest.NewRecorder()
	g.router.ServeHTTP(writer, &http.Request{Method: "GET", Host: "localhost", URL: &url.URL{Path: "/foo"}})
	assert.Equal(t, http.StatusNotFound, writer.Code)
}
