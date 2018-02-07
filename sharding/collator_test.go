package sharding

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
)

type FakeClient struct {
	client *FakeEthClient
}

type FakeEthClient struct{}

type FakeSubscription struct{}

func (ec *FakeEthClient) SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (FakeSubscription, error) {
	return FakeSubscription{}, fmt.Errorf("error, network disconnected!")
}

func TestSubscribeHeaders(t *testing.T) {
	client := &FakeClient{client: &FakeEthClient{}}
	err := subscribeBlockHeaders(client)
	if err != nil {
		t.Errorf("subscribe new headers should work", "no error", err)
	}
}
