package pandora

import (
	"context"
	"github.com/ethereum/go-ethereum/rpc"
)

type PandoraClient struct {
	c *rpc.Client
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

func (oc *PandoraClient) Close() {
	oc.c.Close()
}

func (oc *PandoraClient) PrepareExecutableBlock(ctx context.Context, params *PrepareBlockRequest) (*PrepareBlockResponse, error) {
	var response PrepareBlockResponse
	if err := oc.c.CallContext(ctx, &response, "pandora_prepareExecutableBlock", params); err != nil {
		return nil, err
	}
	return &response, nil
}

func (oc *PandoraClient) InsertExecutableBlock(ctx context.Context, params *InsertBlockRequest) (*InsertBlockResponse, error) {
	var response InsertBlockResponse
	if err := oc.c.CallContext(ctx, &response, "pandora_insertExecutableBlock", params); err != nil {
		return nil, err
	}
	return &response, nil
}
