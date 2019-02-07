package rpc

import (
	"context"
	"testing"

	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestAttestHead(t *testing.T) {
	mockOperationService := &mockOperationService{}
	attesterServer := &AttesterServer{
		operationService: mockOperationService,
	}
	req := &pbp2p.Attestation{
		Data: &pbp2p.AttestationData{
			Slot:                 999,
			Shard:                1,
			ShardBlockRootHash32: []byte{'a'},
		},
	}
	if _, err := attesterServer.AttestHead(context.Background(), req); err != nil {
		t.Errorf("Could not attest head correctly: %v", err)
	}
}
