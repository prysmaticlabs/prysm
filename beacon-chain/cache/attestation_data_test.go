package cache_test

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

func TestAttestationCache_RoundTrip(t *testing.T) {
	ctx := context.Background()
	c := cache.NewAttestationCache()

	req := &pb.AttestationRequest{
		Shard: 0,
		Slot:  1,
	}

	response, err := c.Get(ctx, req)
	if err != nil {
		t.Error(err)
	}

	if response != nil {
		t.Errorf("Empty cache returned an object: %v", response)
	}

	if err := c.MarkInProgress(req); err != nil {
		t.Error(err)
	}

	res := &ethpb.AttestationData{
		Target: &ethpb.Checkpoint{Epoch: 5},
	}

	if err = c.Put(ctx, req, res); err != nil {
		t.Error(err)
	}

	if err := c.MarkNotInProgress(req); err != nil {
		t.Error(err)
	}

	response, err = c.Get(ctx, req)
	if err != nil {
		t.Error(err)
	}

	if !proto.Equal(response, res) {
		t.Error("Expected equal protos to return from cache")
	}
}
