package pandora

import (
	"context"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"reflect"
	"testing"
)

// TestGetShardBlockHeader_Success method checks GetWork method.
func TestGetShardBlockHeader_Success(t *testing.T) {
	// Create a mock server
	server := NewMockPandoraServer()
	defer server.Stop()
	// Create a mock pandora client with in process rpc client
	mockedPandoraClient, err := DialInProcRPCClient(HttpEndpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := mockedPandoraClient.Close()
		require.NoError(t, err)
	}()

	inputBlock := getDummyBlock()
	var response *ShardBlockHeaderResponse
	response, err = mockedPandoraClient.GetShardBlockHeader(context.Background(), types.EmptyRootHash, 1000)
	if err != nil {
		t.Fatal(err)
	}
	// Checks decoding mechanism of incoming response
	if !reflect.DeepEqual(response, &ShardBlockHeaderResponse{
		inputBlock.Hash(),
		inputBlock.Header().ReceiptHash,
		inputBlock.Header(),
		inputBlock.Number().Uint64()}) {
		t.Errorf("incorrect result %#v", response)
	}
}

// TestSubmitShardBlockHeader_Success method checks `eth_submitWork` api
func TestSubmitShardBlockHeader_Success(t *testing.T) {
	// Create a mock server
	server := NewMockPandoraServer()
	defer server.Stop()
	// Create a mock pandora client with in process rpc client
	mockedPandoraClient, err := DialInProcRPCClient(HttpEndpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := mockedPandoraClient.Close()
		require.NoError(t, err)
	}()

	block := getDummyBlock()
	dummySig := [96]byte{}
	response, err := mockedPandoraClient.SubmitShardBlockHeader(context.Background(), block.Nonce(), block.Header().Hash(), dummySig)
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, true, response, "Should OK")
}
