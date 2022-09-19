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
	"github.com/prysmaticlabs/prysm/v3/api/gateway/apimiddleware"
	"github.com/prysmaticlabs/prysm/v3/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
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

	opts := []Option{
		WithRouter(r),
		WithRemoteCert(cert),
		WithAllowedOrigins(origins),
		WithMaxCallRecvMsgSize(size),
		WithApiMiddleware(endpointFactory),
		WithMuxHandler(func(
			_ *apimiddleware.ApiProxyMiddleware,
			_ http.HandlerFunc,
			_ http.ResponseWriter,
			_ *http.Request,
		) {
		}),
	}

	g, err := New(context.Background(), opts...)
	require.NoError(t, err)

	assert.Equal(t, r, g.cfg.router)
	assert.Equal(t, cert, g.cfg.remoteCert)
	require.Equal(t, 1, len(g.cfg.allowedOrigins))
	assert.Equal(t, origins[0], g.cfg.allowedOrigins[0])
	assert.Equal(t, size, g.cfg.maxCallRecvMsgSize)
	assert.Equal(t, endpointFactory, g.cfg.apiMiddlewareEndpointFactory)
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

	opts := []Option{
		WithGatewayAddr(gatewayAddress),
		WithRemoteAddr(selfAddress),
		WithMuxHandler(func(
			_ *apimiddleware.ApiProxyMiddleware,
			_ http.HandlerFunc,
			_ http.ResponseWriter,
			_ *http.Request,
		) {
		}),
	}

	g, err := New(context.Background(), opts...)
	require.NoError(t, err)

	g.Start()
	go func() {
		require.LogsContain(t, hook, "Starting gRPC gateway")
		require.LogsDoNotContain(t, hook, "Starting API middleware")
	}()
	err = g.Stop()
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

	opts := []Option{
		WithGatewayAddr(gatewayAddress),
		WithRemoteAddr(selfAddress),
	}

	g, err := New(context.Background(), opts...)
	require.NoError(t, err)

	writer := httptest.NewRecorder()
	g.cfg.router.ServeHTTP(writer, &http.Request{Method: "GET", Host: "localhost", URL: &url.URL{Path: "/foo"}})
	assert.Equal(t, http.StatusNotFound, writer.Code)
}
