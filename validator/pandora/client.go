package pandora

import (
	"context"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
)

// PandoraClient is a wrapper of rpc client
type PandoraClient struct {
	c *rpc.Client
}

// GetWorkResponseParams contains response params of `eth_gethWork` api
type GetWorkResponseParams struct {
	HeaderHash  common.Hash
	TxReceipt   common.Hash
	Header      *types.Header
	BlockNumber uint64
}

// RpcProgressParams contains response params of `eth_syncing` api
type RpcProgressParams struct {
	StartingBlock hexutil.Uint64
	CurrentBlock  hexutil.Uint64
	HighestBlock  hexutil.Uint64
	PulledStates  hexutil.Uint64
	KnownStates   hexutil.Uint64
}

// Dial creates a new Pandora client.
func Dial(rawurl string) (*PandoraClient, error) {
	return DialContext(context.Background(), rawurl)
}

// DialContext creates a new Pandora client just like Dial
func DialContext(ctx context.Context, rawurl string) (*PandoraClient, error) {
	c, err := rpc.DialContext(ctx, rawurl)
	if err != nil {
		return nil, err
	}
	return NewClient(c), nil
}

// NewClient initialize new pandora client
func NewClient(c *rpc.Client) *PandoraClient {
	return &PandoraClient{c}
}

// Close method closes rpc client.
func (oc *PandoraClient) Close() error {
	if oc.c != nil {
		oc.c.Close()
	}
	return nil
}

// GetWork method calls to pandora client's `eth_getWork` api for getting executable block
// Response structure -
//  - result[0], 32 bytes hex encoded current block header pos-hash
//  - result[1], 32 bytes hex encoded receipt hash for transaction proof
//  - result[2], hex encoded rlp block header
//  - result[3], hex encoded block number
func (oc *PandoraClient) GetWork(ctx context.Context) (*GetWorkResponseParams, error) {
	var response []string
	if err := oc.c.CallContext(ctx, &response, "eth_getWork"); err != nil {
		return nil, errors.Wrap(err, "Got error when calls to eth_getWork api")
	}

	headerHash := common.HexToHash(response[0])
	receiptHash := common.HexToHash(response[1])
	rlpHeader, err := hexutil.Decode(response[2])
	if nil != err {
		return nil, errors.Wrap(err, "Failed to encode hex header")
	}
	header := types.Header{}
	if err := rlp.DecodeBytes(rlpHeader, &header); err != nil {
		return nil, errors.Wrap(err, "Failed to decode header")
	}

	//blockNumberBytes, err := hexutil.Decode(response[3])
	//if err != nil {
	//	return nil, errors.Wrap(err, "Failed to decode block number")
	//}
	//empty4bytes := make([]byte, 8)
	//blockNumber := binary.BigEndian.Uint64(append(blockNumberBytes[:8], empty4bytes...))

	return &GetWorkResponseParams{
		HeaderHash:  headerHash,
		TxReceipt:   receiptHash,
		Header:      &header,
		BlockNumber: header.Number.Uint64(),
	}, nil
}

// SubmitWork methods call to pandora client's `eth_submitWork` api
func (oc *PandoraClient) SubmitWork(ctx context.Context, blockNonce uint64, headerHash common.Hash,
	sig [32]byte) (bool, error) {

	nonecHex := types.EncodeNonce(blockNonce)
	headerHashHex := headerHash.Hex()
	sigHex := common.BytesToHash(sig[:])

	var status bool
	if err := oc.c.CallContext(ctx, &status, "eth_submitWork", nonecHex, headerHashHex, sigHex); err != nil {
		return false, errors.Wrap(err, "Got error when calls to eth_submitWork api")
	}
	return status, nil
}

// SyncProgress retrieves the current progress of the sync algorithm. If there's
// no sync currently running, it returns nil.
func (ec *PandoraClient) SyncProgress(ctx context.Context) (*RpcProgressParams, error) {
	var raw json.RawMessage
	if err := ec.c.CallContext(ctx, &raw, "eth_syncing"); err != nil {
		return nil, err
	}
	// Handle the possible response types
	var syncing bool
	if err := json.Unmarshal(raw, &syncing); err == nil {
		return nil, nil // Not syncing (always false)
	}
	var progress *RpcProgressParams
	if err := json.Unmarshal(raw, &progress); err != nil {
		return nil, err
	}
	return &RpcProgressParams{
		StartingBlock: progress.StartingBlock,
		CurrentBlock:  progress.CurrentBlock,
		HighestBlock:  progress.HighestBlock,
		PulledStates:  progress.PulledStates,
		KnownStates:   progress.KnownStates,
	}, nil
}
