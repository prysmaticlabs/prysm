//go:build use_beacon_api
// +build use_beacon_api

package beacon_api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	rpcmiddleware "github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestWaitForChainStart_ValidGenesis(t *testing.T) {
	server := httptest.NewServer(createGenesisHandler(&rpcmiddleware.GenesisResponse_GenesisJson{
		GenesisTime:           "1234",
		GenesisValidatorsRoot: "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
	}))
	defer server.Close()

	validatorClient := NewBeaconApiValidatorClient(server.URL, time.Second*5)

	resp, err := validatorClient.WaitForChainStart(context.Background(), &emptypb.Empty{})
	assert.NoError(t, err)

	require.NotNil(t, resp)
	assert.Equal(t, true, resp.Started)
	assert.Equal(t, uint64(1234), resp.GenesisTime)

	expectedRoot, err := hexutil.Decode("0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2")
	require.NoError(t, err)
	assert.DeepEqual(t, expectedRoot, resp.GenesisValidatorsRoot)
}

func TestWaitForChainStart_NilData(t *testing.T) {
	server := httptest.NewServer(createGenesisHandler(nil))
	defer server.Close()

	validatorClient := NewBeaconApiValidatorClient(server.URL, time.Second*5)
	_, err := validatorClient.WaitForChainStart(context.Background(), &emptypb.Empty{})
	assert.ErrorContains(t, "failed to get genesis data", err)
}

func TestWaitForChainStart_InvalidTime(t *testing.T) {
	server := httptest.NewServer(createGenesisHandler(&rpcmiddleware.GenesisResponse_GenesisJson{
		GenesisTime:           "foo",
		GenesisValidatorsRoot: "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
	}))
	defer server.Close()

	validatorClient := NewBeaconApiValidatorClient(server.URL, time.Second*5)
	_, err := validatorClient.WaitForChainStart(context.Background(), &emptypb.Empty{})
	assert.ErrorContains(t, "failed to parse genesis time", err)
}

func TestWaitForChainStart_EmptyTime(t *testing.T) {
	server := httptest.NewServer(createGenesisHandler(&rpcmiddleware.GenesisResponse_GenesisJson{
		GenesisTime:           "",
		GenesisValidatorsRoot: "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
	}))
	defer server.Close()

	validatorClient := NewBeaconApiValidatorClient(server.URL, time.Second*5)
	_, err := validatorClient.WaitForChainStart(context.Background(), &emptypb.Empty{})
	assert.ErrorContains(t, "failed to parse genesis time", err)
}

func TestWaitForChainStart_InvalidRoot(t *testing.T) {
	server := httptest.NewServer(createGenesisHandler(&rpcmiddleware.GenesisResponse_GenesisJson{
		GenesisTime:           "1234",
		GenesisValidatorsRoot: "0xzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
	}))
	defer server.Close()

	validatorClient := NewBeaconApiValidatorClient(server.URL, time.Second*5)
	_, err := validatorClient.WaitForChainStart(context.Background(), &emptypb.Empty{})
	assert.ErrorContains(t, "invalid genesis validators root", err)
}

func TestWaitForChainStart_EmptyRoot(t *testing.T) {
	server := httptest.NewServer(createGenesisHandler(&rpcmiddleware.GenesisResponse_GenesisJson{
		GenesisTime:           "1234",
		GenesisValidatorsRoot: "",
	}))
	defer server.Close()

	validatorClient := NewBeaconApiValidatorClient(server.URL, time.Second*5)
	_, err := validatorClient.WaitForChainStart(context.Background(), &emptypb.Empty{})
	assert.ErrorContains(t, "invalid genesis validators root", err)
}

func TestWaitForChainStart_InvalidJsonGenesis(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("foo"))
		require.NoError(t, err)
	}))
	defer server.Close()

	validatorClient := NewBeaconApiValidatorClient(server.URL, time.Second*5)
	_, err := validatorClient.WaitForChainStart(context.Background(), &emptypb.Empty{})
	assert.ErrorContains(t, "failed to get genesis data", err)
}

func TestWaitForChainStart_InternalServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(internalServerErrHandler))
	defer server.Close()

	validatorClient := NewBeaconApiValidatorClient(server.URL, time.Second*5)
	_, err := validatorClient.WaitForChainStart(context.Background(), &emptypb.Empty{})
	assert.ErrorContains(t, "500: Internal server error", err)
}

func TestWaitForChainStart_NotFoundErrorContextCancelled(t *testing.T) {
	// WaitForChainStart blocks until the error is not 404, but it needs to listen to context cancellations
	server := httptest.NewServer(http.HandlerFunc(notFoundErrHandler))
	defer server.Close()

	validatorClient := NewBeaconApiValidatorClient(server.URL, time.Second*5)

	// Create a context that can be canceled
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel the context after 1 second
	go func(ctx context.Context) {
		time.Sleep(time.Second)
		cancel()
	}(ctx)

	_, err := validatorClient.WaitForChainStart(ctx, &emptypb.Empty{})
	assert.ErrorContains(t, "context canceled", err)
}

// This test makes sure that we handle even errors not specified in the Beacon API spec
func TestWaitForChainStart_UnknownError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(invalidErr999Handler))
	defer server.Close()

	validatorClient := NewBeaconApiValidatorClient(server.URL, time.Second*5)
	_, err := validatorClient.WaitForChainStart(context.Background(), &emptypb.Empty{})
	assert.ErrorContains(t, "999: Invalid error", err)
}

// Make sure that we fail gracefully if the error json is not valid json
func TestWaitForChainStart_InvalidJsonError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(invalidJsonErrHandler))
	defer server.Close()

	validatorClient := NewBeaconApiValidatorClient(server.URL, time.Second*5)
	_, err := validatorClient.WaitForChainStart(context.Background(), &emptypb.Empty{})
	assert.ErrorContains(t, "failed to get genesis data", err)
}

func TestWaitForChainStart_Timeout(t *testing.T) {
	server := httptest.NewServer(createGenesisHandler(&rpcmiddleware.GenesisResponse_GenesisJson{
		GenesisTime:           "1234",
		GenesisValidatorsRoot: "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
	}))
	defer server.Close()

	validatorClient := NewBeaconApiValidatorClient(server.URL, 1)
	_, err := validatorClient.WaitForChainStart(context.Background(), &emptypb.Empty{})
	assert.ErrorContains(t, "failed to get genesis data", err)
}
