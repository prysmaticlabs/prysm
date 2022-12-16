package withdrawal

import (
	"flag"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/urfave/cli/v2"
)

func TestCallWithdrawalEndpoint(t *testing.T) {
	baseurl := "127.0.0.1:3500"
	l, err := net.Listen("tcp", baseurl)
	require.NoError(t, err)
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
	}))
	err = srv.Listener.Close()
	require.NoError(t, err)
	srv.Listener = l
	srv.Start()
	defer srv.Close()
	hook := logtest.NewGlobal()

	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String("beacon-node-host", "http://"+baseurl, "")
	set.String("file", "./testdata/change-operations.json", "")
	assert.NoError(t, set.Set("beacon-node-host", "http://"+baseurl))
	assert.NoError(t, set.Set("file", "./testdata/change-operations.json"))
	cliCtx := cli.NewContext(&app, set, nil)

	content := []byte("0xa94f5374fce5edbc8e2a8697c15331677e6ebf0b")
	tmpfile, err := os.CreateTemp("", "content")
	require.NoError(t, err)
	defer func() {
		err := os.Remove(tmpfile.Name())
		require.NoError(t, err)
	}()

	_, err = tmpfile.Write(content)
	require.NoError(t, err)

	_, err = tmpfile.Seek(0, 0)
	require.NoError(t, err)
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }() // Restore original Stdin

	os.Stdin = tmpfile
	err = setWithdrawalAddress(cliCtx, os.Stdin)
	require.NoError(t, err)

	assert.LogsContain(t, hook, "Successfully published message to update withdrawal address.")
}
