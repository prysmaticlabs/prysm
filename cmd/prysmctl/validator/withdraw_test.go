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
	set.String("beacon-node-host", baseurl, "")
	set.String("path", file, "")
	set.Bool("confirm", true, "")
	set.Bool("accept-terms-of-use", true, "")
	assert.NoError(t, set.Set("beacon-node-host", baseurl))
	assert.NoError(t, set.Set("path", file))
	cliCtx := cli.NewContext(&app, set, nil)

	err = setWithdrawalAddresses(cliCtx)
	require.NoError(t, err)

	assert.LogsContain(t, hook, "Successfully published")
}

func TestCallWithdrawalEndpoint_Mutiple(t *testing.T) {
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
	set.String("beacon-node-host", baseurl, "")
	set.String("path", file, "")
	set.Bool("confirm", true, "")
	set.Bool("accept-terms-of-use", true, "")
	assert.NoError(t, set.Set("beacon-node-host", baseurl))
	assert.NoError(t, set.Set("path", file))
	cliCtx := cli.NewContext(&app, set, nil)

	err = setWithdrawalAddresses(cliCtx)
	require.NoError(t, err)
	assert.LogsContain(t, hook, "Successfully published")
	assert.LogsContain(t, hook, "to update 2 withdrawal")
	assert.LogsContain(t, hook, "All (total:2) signed withdrawal messages were found in the pool.")
	assert.LogsDoNotContain(t, hook, "Set withdrawal address message not found in the node's operations pool.")
}

func TestCallWithdrawalEndpoint_Mutiple_stakingcli(t *testing.T) {
	stakingcliFile := "./testdata/staking-cli-change-operations-multiple.json"
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
	set.String("beacon-node-host", baseurl, "")
	set.String("path", stakingcliFile, "")
	set.Bool("confirm", true, "")
	set.Bool("accept-terms-of-use", true, "")
	assert.NoError(t, set.Set("beacon-node-host", baseurl))
	assert.NoError(t, set.Set("path", stakingcliFile))
	cliCtx := cli.NewContext(&app, set, nil)

	err = setWithdrawalAddresses(cliCtx)
	require.NoError(t, err)
	assert.LogsContain(t, hook, "Successfully published")
	assert.LogsContain(t, hook, "to update 2 withdrawal")
	assert.LogsContain(t, hook, "All (total:2) signed withdrawal messages were found in the pool.")
	assert.LogsDoNotContain(t, hook, "Set withdrawal address message not found in the node's operations pool.")
}

func TestCallWithdrawalEndpoint_Mutiple_notfound(t *testing.T) {
	respFile := "./testdata/change-operations-multiple_notfound.json"
	file := "./testdata/change-operations-multiple.json"
	baseurl := "127.0.0.1:3500"
	l, err := net.Listen("tcp", baseurl)
	require.NoError(t, err)
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			b, err := os.ReadFile(filepath.Clean(respFile))
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
	set.String("beacon-node-host", baseurl, "")
	set.String("path", file, "")
	set.Bool("confirm", true, "")
	set.Bool("accept-terms-of-use", true, "")
	assert.NoError(t, set.Set("beacon-node-host", baseurl))
	assert.NoError(t, set.Set("path", file))
	cliCtx := cli.NewContext(&app, set, nil)

	err = setWithdrawalAddresses(cliCtx)
	require.NoError(t, err)
	assert.LogsContain(t, hook, "Successfully published")
	assert.LogsContain(t, hook, "to update 2 withdrawal")
	assert.LogsContain(t, hook, "Set withdrawal address message not found in the node's operations pool.")
	assert.LogsContain(t, hook, "Please check before resubmitting. Set withdrawal address messages that were not found in the pool may have been already included into a block.")
	assert.LogsDoNotContain(t, hook, "Set withdrawal address message found in the node's operations pool.")
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
	set.String("beacon-node-host", baseurl, "")
	set.String("path", tmpfile.Name(), "")
	set.Bool("confirm", true, "")
	set.Bool("accept-terms-of-use", true, "")
	assert.NoError(t, set.Set("beacon-node-host", baseurl))
	assert.NoError(t, set.Set("path", tmpfile.Name()))
	cliCtx := cli.NewContext(&app, set, nil)
	err = setWithdrawalAddresses(cliCtx)
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
	set.String("beacon-node-host", baseurl, "")
	set.String("path", file, "")
	set.Bool("confirm", true, "")
	set.Bool("accept-terms-of-use", true, "")
	assert.NoError(t, set.Set("beacon-node-host", baseurl))
	assert.NoError(t, set.Set("path", file))
	cliCtx := cli.NewContext(&app, set, nil)

	err = setWithdrawalAddresses(cliCtx)
	assert.ErrorContains(t, "POST error", err)

	assert.LogsContain(t, hook, "Could not validate SignedBLSToExecutionChange")
}

func TestVerifyWithdrawal_Mutiple(t *testing.T) {
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
	set.String("beacon-node-host", baseurl, "")
	set.String("path", file, "")
	set.Bool("confirm", true, "")
	set.Bool("accept-terms-of-use", true, "")
	set.Bool("verify-only", true, "")
	assert.NoError(t, set.Set("beacon-node-host", baseurl))
	assert.NoError(t, set.Set("path", file))
	cliCtx := cli.NewContext(&app, set, nil)

	err = verifyWithdrawalsInPool(cliCtx)
	require.NoError(t, err)
	assert.LogsContain(t, hook, "All (total:2) signed withdrawal messages were found in the pool.")
	assert.LogsDoNotContain(t, hook, "set withdrawal address message not found in the node's operations pool.")
}
