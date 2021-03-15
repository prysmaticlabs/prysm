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
	"math/big"
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

// getDummyEncodedExtraData prepares rlp encoded extra data
func getDummyEncodedExtraData() (*ExtraData, []byte, error) {
	extraData := ExtraData{
		Slot:          98,
		Epoch:         3,
		ProposerIndex: 23,
	}
	extraDataByte, err := rlp.EncodeToBytes(extraData)
	return &extraData, extraDataByte, err
}

// getDummyBlock method creates a brand new block with extraData
func getDummyBlock() *types.Block {
	_, extraDataByte, err := getDummyEncodedExtraData()
	if err != nil {
		return nil
	}
	block := types.NewBlock(&types.Header{
		ParentHash:  types.EmptyRootHash,
		UncleHash:   types.EmptyUncleHash,
		Coinbase:    common.HexToAddress("8888f1f195afa192cfee860698584c030f4c9db1"),
		Root:        common.HexToHash("ef1552a40b7165c3cd773806b9e0c165b75356e0314bf0706f279c729f51e017"),
		TxHash:      types.EmptyRootHash,
		ReceiptHash: types.EmptyRootHash,
		Difficulty:  big.NewInt(131072),
		Number:      big.NewInt(314),
		GasLimit:    uint64(3141592),
		GasUsed:     uint64(21000),
		Time:        uint64(1426516743),
		Extra:       extraDataByte,
		MixDigest:   common.HexToHash("bd4472abb6659ebe3ee06ee4d7b72a00a9f4d001caca51342001075469aff498"),
		Nonce:       types.BlockNonce{0x01, 0x02, 0x03},
	}, nil, nil, nil, nil)

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

// Syncing is a mock api which returns a dummy chain info
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
