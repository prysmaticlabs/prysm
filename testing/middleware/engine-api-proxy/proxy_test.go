package proxy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/rpc"
	pb "github.com/prysmaticlabs/prysm/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestProxy(t *testing.T) {
	t.Run("fails to proxy if destination is down", func(t *testing.T) {
		logger := logrus.New()
		hook := logTest.NewLocal(logger)
		ctx := context.Background()
		proxy, err := New(
			WithDestinationAddress("http://localhost:43239"), // Nothing running at destination server.
			WithLogger(logger),
		)
		require.NoError(t, err)
		go func() {
			if err := proxy.Start(ctx); err != nil {
				t.Log(err)
			}
		}()

		rpcClient, err := rpc.DialHTTP("http://" + proxy.Address())
		require.NoError(t, err)

		err = rpcClient.CallContext(ctx, nil, "someEngineMethod")
		require.ErrorContains(t, "EOF", err)

		// Expect issues when reaching destination server.
		require.LogsContain(t, hook, "Could not forward request to destination server")
	})
	t.Run("properly proxies request/response", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		wantDestinationResponse := &pb.ForkchoiceState{
			HeadBlockHash:      []byte("foo"),
			SafeBlockHash:      []byte("bar"),
			FinalizedBlockHash: []byte("baz"),
		}
		srv := destinationServerSetup(t, wantDestinationResponse)
		defer srv.Close()

		// Destination address server responds to JSON-RPC requests.
		proxy, err := New(
			WithDestinationAddress(srv.URL),
		)
		require.NoError(t, err)
		go func() {
			if err := proxy.Start(ctx); err != nil {
				t.Log(err)
			}
		}()

		// Dials the proxy.
		rpcClient, err := rpc.DialHTTP("http://" + proxy.Address())
		require.NoError(t, err)

		// Expect the result from the proxy is the same as that one returned
		// by the destination address.
		proxyResult := &pb.ForkchoiceState{}
		err = rpcClient.CallContext(ctx, proxyResult, "someEngineMethod")
		require.NoError(t, err)
		require.DeepEqual(t, wantDestinationResponse, proxyResult)
	})
}

func TestProxy_CustomInterceptors(t *testing.T) {
	t.Run("only intercepts engine API methods", func(t *testing.T) {

	})
	t.Run("only intercepts if trigger function returns true", func(t *testing.T) {
		//proxyNode.AddInterceptor(func(reqBytes []byte, w http.ResponseWriter, r *http.Request) bool {
		//	currSlot := slots.CurrentSlot(uint64(genesis.GenesisTime.AsTime().Unix()))
		//	if currSlot < params.BeaconConfig().SlotsPerEpoch.Mul(uint64(9)) {
		//		return false
		//	}
		//	if currSlot >= params.BeaconConfig().SlotsPerEpoch.Mul(uint64(10)) {
		//		return false
		//	}
		//	return proxyNode.SyncingInterceptor()(reqBytes, w, r)
		//})
	})
	t.Run("triggers interceptor response correctly", func(t *testing.T) {

	})
}

func Test_isEngineAPICall(t *testing.T) {
	t.Run("naked array returns false", func(t *testing.T) {

	})
	t.Run("non-engine call returns false", func(t *testing.T) {

	})
	t.Run("engine call returns true", func(t *testing.T) {

	})
}

func destinationServerSetup(t *testing.T, response interface{}) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		defer func() {
			require.NoError(t, r.Body.Close())
		}()
		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  response,
		}
		err := json.NewEncoder(w).Encode(resp)
		require.NoError(t, err)
	}))
}
