package httprest

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"github.com/urfave/cli/v2"
)

func TestServer_StartStop(t *testing.T) {
	hook := logTest.NewGlobal()

	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	ctx := cli.NewContext(&app, set, nil)

	port := ctx.Int(flags.HTTPServerPort.Name)
	host := ctx.String(flags.HTTPServerHost.Name)
	address := fmt.Sprintf("%s:%d", host, port)

	opts := []Option{
		WithHTTPAddr(address),
		WithMuxHandler(func(
			_ http.HandlerFunc,
			_ http.ResponseWriter,
			_ *http.Request,
		) {
		}),
		WithRouter(http.NewServeMux()),
	}

	g, err := New(context.Background(), opts...)
	require.NoError(t, err)

	g.Start()
	go func() {
		require.LogsContain(t, hook, "Starting HTTP server")
		require.LogsDoNotContain(t, hook, "Starting API middleware")
	}()
	err = g.Stop()
	require.NoError(t, err)
}

func TestServer_NilHandler_NotFoundHandlerRegistered(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	ctx := cli.NewContext(&app, set, nil)

	port := ctx.Int(flags.HTTPServerPort.Name)
	host := ctx.String(flags.HTTPServerHost.Name)
	address := fmt.Sprintf("%s:%d", host, port)

	opts := []Option{
		WithHTTPAddr(address),
		WithRouter(http.NewServeMux()),
	}

	g, err := New(context.Background(), opts...)
	require.NoError(t, err)

	writer := httptest.NewRecorder()
	g.cfg.router.ServeHTTP(writer, &http.Request{Method: "GET", Host: "localhost", URL: &url.URL{Path: "/foo"}})
	assert.Equal(t, http.StatusNotFound, writer.Code)
}
