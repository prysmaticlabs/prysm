package pandora

import (
	"context"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"reflect"
	"testing"
)

// TestGetWork_OK method checks GetWork method.
func TestGetWork_OK(t *testing.T) {
	// Create a mock server
	server := NewMockPandoraServer()
	defer server.Stop()
	// Create a mock pandora client with in process rpc client
	mockedPandoraClient, err := DialInProcRPCClient(HttpEndpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer mockedPandoraClient.Close()

	inputBlock := getDummyBlock()
	var response *GetWorkResponseParams
	response, err = mockedPandoraClient.GetWork(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// Checks decoding mechanism of incoming response
	if !reflect.DeepEqual(response, &GetWorkResponseParams{
		inputBlock.Hash(),
		inputBlock.Header().ReceiptHash,
		inputBlock.Header(),
		inputBlock.Number().Uint64()}) {
		t.Errorf("incorrect result %#v", response)
	}
}

// TestSubmitWork_OK method checks `eth_submitWork` api
func TestSubmitWork_OK(t *testing.T) {
	// Create a mock server
	server := NewMockPandoraServer()
	defer server.Stop()
	// Create a mock pandora client with in process rpc client
	mockedPandoraClient, err := DialInProcRPCClient(HttpEndpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer mockedPandoraClient.Close()

	block := getDummyBlock()
	dummySig := [32]byte{}
	response, err := mockedPandoraClient.SubmitWork(context.Background(), block.Nonce(), block.Header().Hash(), dummySig)
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, true, response, "Should OK")
}
