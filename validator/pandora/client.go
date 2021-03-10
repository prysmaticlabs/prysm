package pandora

import (
	"context"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

type PandoraClient struct {
	c *rpc.Client
}

type rpcProgress struct {
	StartingBlock hexutil.Uint64
	CurrentBlock  hexutil.Uint64
	HighestBlock  hexutil.Uint64
	PulledStates  hexutil.Uint64
	KnownStates   hexutil.Uint64
}

func Dial(rawurl string) (*PandoraClient, error) {
	return DialContext(context.Background(), rawurl)
}

func DialContext(ctx context.Context, rawurl string) (*PandoraClient, error) {
	c, err := rpc.DialContext(ctx, rawurl)
	if err != nil {
		return nil, err
	}
	return NewClient(c), nil
}

func NewClient(c *rpc.Client) *PandoraClient {
	return &PandoraClient{c}
}

func (oc *PandoraClient) Close() error {
	oc.c.Close()
	return nil
}

func (oc *PandoraClient) PrepareExecutableBlock(ctx context.Context,
	params *PrepareBlockRequest) (*PrepareBlockResponse, error) {
	var response PrepareBlockResponse
	if err := oc.c.CallContext(ctx, &response, "pandora_prepareExecutableBlock", params); err != nil {
		return nil, err
	}
	return &response, nil
}

func (oc *PandoraClient) InsertExecutableBlock(ctx context.Context,
	params *InsertBlockRequest) (*InsertBlockResponse, error) {
	var response InsertBlockResponse
	if err := oc.c.CallContext(ctx, &response, "pandora_insertExecutableBlock", params); err != nil {
		return nil, err
	}
	return &response, nil
}

// SyncProgress retrieves the current progress of the sync algorithm. If there's
// no sync currently running, it returns nil.
func (ec *PandoraClient) SyncProgress(ctx context.Context) (*rpcProgress, error) {
	var raw json.RawMessage
	if err := ec.c.CallContext(ctx, &raw, "eth_syncing"); err != nil {
		return nil, err
	}
	// Handle the possible response types
	var syncing bool
	if err := json.Unmarshal(raw, &syncing); err == nil {
		return nil, nil // Not syncing (always false)
	}
	var progress *rpcProgress
	if err := json.Unmarshal(raw, &progress); err != nil {
		return nil, err
	}
	return &rpcProgress{
		StartingBlock: progress.StartingBlock,
		CurrentBlock:  progress.CurrentBlock,
		HighestBlock:  progress.HighestBlock,
		PulledStates:  progress.PulledStates,
		KnownStates:   progress.KnownStates,
	}, nil
}
