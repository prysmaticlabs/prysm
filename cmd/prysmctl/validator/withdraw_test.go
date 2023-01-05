package validator

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
	set.String("path", file, "")
	assert.NoError(t, set.Set("beacon-node-host", "http://"+baseurl))
	assert.NoError(t, set.Set("path", file))
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
	set.String("path", file, "")
	set.Bool("confirm", true, "")
	assert.NoError(t, set.Set("beacon-node-host", "http://"+baseurl))
	assert.NoError(t, set.Set("path", file))
	cliCtx := cli.NewContext(&app, set, nil)

	err = setWithdrawalAddresses(cliCtx, os.Stdin)
	require.NoError(t, err)
	assert.LogsContain(t, hook, "Successfully published")
	assert.LogsContain(t, hook, "to update 2 withdrawal")
	assert.LogsContain(t, hook, "set withdrawal address message was found in the node's operations pool.")
	assert.LogsDoNotContain(t, hook, "set withdrawal address message not found in the node's operations pool.")
}

func TestCallWithdrawalEndpoint_Empty(t *testing.T) {
	baseurl := "127.0.0.1:3500"
	content := []byte("[]")
	tmpfile, err := os.CreateTemp("./testdata", "*.json")
	require.NoError(t, err)
	_, err = tmpfile.Write(content)
	require.NoError(t, err)
	defer func() {
		err := os.Remove(tmpfile.Name())
		require.NoError(t, err)
	}()
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String("beacon-node-host", "http://"+baseurl, "")
	set.String("path", tmpfile.Name(), "")
	assert.NoError(t, set.Set("beacon-node-host", "http://"+baseurl))
	assert.NoError(t, set.Set("path", tmpfile.Name()))
	cliCtx := cli.NewContext(&app, set, nil)
	err = setWithdrawalAddresses(cliCtx, os.Stdin)
	assert.ErrorContains(t, "the list of signed requests is empty", err)
}

func TestCallWithdrawalEndpoint_Errors(t *testing.T) {
	file := "./testdata/change-operations.json"
	baseurl := "127.0.0.1:3500"
	l, err := net.Listen("tcp", baseurl)
	require.NoError(t, err)
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(&apimiddleware.IndexedVerificationFailureErrorJson{
			Failures: []*apimiddleware.SingleIndexedVerificationFailureJson{
				{Index: 0, Message: "Could not validate SignedBLSToExecutionChange"},
			},
		})
		require.NoError(t, err)
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
	set.String("path", file, "")
	assert.NoError(t, set.Set("beacon-node-host", "http://"+baseurl))
	assert.NoError(t, set.Set("path", file))
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
	assert.ErrorContains(t, "POST error", err)

	assert.LogsContain(t, hook, "Could not validate SignedBLSToExecutionChange")
}
