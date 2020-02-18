package kv

import (
	"context"
	"flag"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/urfave/cli"
)

func TestChainHead(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(app, set, nil))
	ctx := context.Background()

	tests := []struct {
		head *ethpb.ChainHead
	}{
		{
			head: &ethpb.ChainHead{
				HeadSlot:       20,
				HeadEpoch:      20,
				FinalizedSlot:  10,
				FinalizedEpoch: 10,
				JustifiedSlot:  10,
				JustifiedEpoch: 10,
			},
		},
		{
			head: &ethpb.ChainHead{
				HeadSlot: 1,
			},
		},
		{
			head: &ethpb.ChainHead{
				HeadBlockRoot: make([]byte, 32),
			},
		},
	}

	for _, tt := range tests {
		if err := db.SaveChainHead(ctx, tt.head); err != nil {
			t.Fatal(err)
		}
		head, err := db.ChainHead(ctx)
		if err != nil {
			t.Fatalf("failed to get block: %v", err)
		}
		if head == nil || !proto.Equal(head, tt.head) {
			t.Errorf("Expected %v, got %v", tt.head, head)
		}
	}
}
