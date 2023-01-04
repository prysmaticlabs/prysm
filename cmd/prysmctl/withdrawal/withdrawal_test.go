package withdrawal

import (
	"encoding/json"
	"flag"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/urfave/cli/v2"
)

func TestCallWithdrawalEndpoint(t *testing.T) {
	file := "./testdata/change-operations.json"
	baseurl := "127.0.0.1:3500"
	l, err := net.Listen("tcp", baseurl)
	require.NoError(t, err)
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			b, err := os.ReadFile(filepath.Clean(file))
			require.NoError(t, err)
			var to []*apimiddleware.SignedBLSToExecutionChangeJson
			err = json.Unmarshal(b, &to)
			require.NoError(t, err)
			err = json.NewEncoder(w).Encode(&apimiddleware.BLSToExecutionChangesPoolResponseJson{
				Data: to,
			})
			require.NoError(t, err)
		}
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
	set.String("file", file, "")
	assert.NoError(t, set.Set("beacon-node-host", "http://"+baseurl))
	assert.NoError(t, set.Set("file", file))
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
	err = setWithdrawalAddresses(cliCtx, os.Stdin)
	require.NoError(t, err)

	assert.LogsContain(t, hook, "Successfully published")
}

func TestCallWithdrawalEndpointMutiple(t *testing.T) {
	file := "./testdata/change-operations-multiple.json"
	baseurl := "127.0.0.1:3500"
	l, err := net.Listen("tcp", baseurl)
	require.NoError(t, err)
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			b, err := os.ReadFile(filepath.Clean(file))
			require.NoError(t, err)
			var to []*apimiddleware.SignedBLSToExecutionChangeJson
			err = json.Unmarshal(b, &to)
			require.NoError(t, err)
			err = json.NewEncoder(w).Encode(&apimiddleware.BLSToExecutionChangesPoolResponseJson{
				Data: to,
			})
			require.NoError(t, err)
		}
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
	set.String("file", file, "")
	set.Bool("skip-prompts", true, "")
	assert.NoError(t, set.Set("beacon-node-host", "http://"+baseurl))
	assert.NoError(t, set.Set("file", file))
	cliCtx := cli.NewContext(&app, set, nil)

	err = setWithdrawalAddresses(cliCtx, os.Stdin)
	require.NoError(t, err)

	assert.LogsContain(t, hook, "Successfully published")
	assert.LogsContain(t, hook, "to update 2 withdrawal")
	assert.LogsContain(t, hook, "validator index: 0 with set withdrawal address: 0x0xa94f5374fce5edbc8e2a8697c15331677e6ebf0b")
	assert.LogsContain(t, hook, "validator index: 1 with set withdrawal address: 0x0xa94f5374fce5edbc8e2a8697c15331677e6ebf0b")
}
