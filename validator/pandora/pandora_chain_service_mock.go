package pandora

import (
	"context"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	"math/big"
)

var HttpEndpoint = "http://127.0.0.1:4045"

type mockPandoraService struct{}

// ConnectPandoraService method creates dummy connection with pandora chain
func ConnectPandoraService(pandoraService *Service) {
	pandoraService.connected = true
	pandoraService.isRunning = true
}

// NewMockPandoraServer method mock pandora chain apis
func NewMockPandoraServer() *rpc.Server {
	server := rpc.NewServer()
	if err := server.RegisterName("eth", new(mockPandoraService)); err != nil {
		panic(err)
	}
	return server
}

// DialInProcRPCClient method initializes in process rpc client with mocked pandora chain server.
func DialInProcRPCClient(endpoint string) (*PandoraClient, error) {
	server := NewMockPandoraServer()
	rpcClient := rpc.DialInProc(server)
	pandoraClient := NewClient(rpcClient)

	return pandoraClient, nil
}

// DialRPCClient method initialize rpc client with real pandora chain server over http
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
func (api *mockPandoraService) SubmitWorkBLS(nonce types.BlockNonce, hash common.Hash, blsSignature string) bool {
	block := getDummyBlock()
	if block.Hash() != hash {
		return false
	}
	if len(blsSignature) != 194 {
		return false
	}
	return true
}

// Syncing is a mock api which returns a dummy chain info
func (api *mockPandoraService) Syncing(ctx context.Context) (*ShardChainSyncResponse, error) {
	return &ShardChainSyncResponse{
		StartingBlock: 12,
		CurrentBlock:  192,
		HighestBlock:  200,
		PulledStates:  12,
		KnownStates:   12,
	}, nil
}
