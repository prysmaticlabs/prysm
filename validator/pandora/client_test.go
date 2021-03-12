package pandora

import (
	"context"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"reflect"
	"testing"
)

var HttpEndpoint = "http://127.0.0.1:4045"

type mockPandoraService struct{}

func NewMockPandoraServer() *rpc.Server {
	server := rpc.NewServer()
	if err := server.RegisterName("eth", new(mockPandoraService)); err != nil {
		panic(err)
	}
	return server
}

func DialInProcRPCClient(endpoint string) (*PandoraClient, error) {
	server := NewMockPandoraServer()
	rpcClient := rpc.DialInProc(server)
	pandoraClient := NewClient(rpcClient)

	return pandoraClient, nil
}

func DialRPCClient(endpoint string) (*PandoraClient, error) {
	rpcClient, err := rpc.Dial(endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not dial node")
	}
	pandoraClient := NewClient(rpcClient)
	return pandoraClient, nil
}

func getDummyBlock() types.Block {
	blockEnc := common.FromHex("f90260f901f9a083cafc574e1f51ba9dc0568fc617a08ea2429fb384059c972f13b19fa1c8dd55a01dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347948888f1f195afa192cfee860698584c030f4c9db1a0ef1552a40b7165c3cd773806b9e0c165b75356e0314bf0706f279c729f51e017a05fe50b260da6308036625b850b5d6ced6d0a9f814c0688bc91ffb7b7a3a54b67a0bc37d79753ad738a6dac4921e57392f145d8887476de3f783dfa7edae9283e52b90100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000008302000001832fefd8825208845506eb0780a0bd4472abb6659ebe3ee06ee4d7b72a00a9f4d001caca51342001075469aff49888a13a5a8c8f2bb1c4f861f85f800a82c35094095e7baea6a6c7c4c2dfeb977efac326af552d870a801ba09bea4c4daac7c7c52e093e6a4c35dbbcf8856f1af7b059ba20253e70848d094fa08a8fae537ce25ed8cb5af9adac3f141af69bd515bd2ba031522df09b97dd72b1c0")
	var block types.Block
	if err := rlp.DecodeBytes(blockEnc, &block); err != nil {
		return types.Block{}
	}
	return block
}

// GetWork is a mock api which returns a work package for external miner.
// The work package consists of 3 strings:
//   result[0] - 32 bytes hex encoded current block header pow-hash
//   result[1] - 32 bytes hex encoded seed hash used for DAG
//   result[2] - 32 bytes hex encoded boundary condition ("target"), 2^256/difficulty
//   result[3] - hex encoded block number
func (api *mockPandoraService) GetWork() ([4]string, error) {
	block := getDummyBlock()
	var response [4]string
	rlpHeader, _ := rlp.EncodeToBytes(block.Header())

	response[0] = block.Hash().Hex()
	response[1] = block.Header().ReceiptHash.Hex()
	response[2] = hexutil.Encode(rlpHeader)
	response[3] = hexutil.Encode(block.Header().Number.Bytes())

	return response, nil
}

// SubmitWork is a mock api which returns a boolean status
func (api *mockPandoraService) SubmitWork(nonce types.BlockNonce, hash, digest common.Hash) bool {
	block := getDummyBlock()
	if block.Hash() != hash {
		return false
	}
	if len(digest) != 32 {
		return false
	}
	return true
}

func (api *mockPandoraService) Syncing(ctx context.Context) (*RpcProgressParams, error) {
	return &RpcProgressParams{
		StartingBlock: 12,
		CurrentBlock:  192,
		HighestBlock:  200,
		PulledStates:  12,
		KnownStates:   12,
	}, nil
}

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

//
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
