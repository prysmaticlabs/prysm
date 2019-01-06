package stateutils_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/stateutils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestValidatorIndexMap(t *testing.T) {
	state := &pb.BeaconState{
		ValidatorRegistry: []*pb.ValidatorRecord{
			&pb.ValidatorRecord{
				Pubkey: []byte("zero"),
			},
			&pb.ValidatorRecord{
				Pubkey: []byte("one"),
			},
		},
	}

	tests := []struct {
		key [32]byte
		val int
		ok  bool
	}{
		{
			key: stateutils.BytesToBytes32([]byte("zero")),
			val: 0,
			ok:  true,
		}, {
			key: stateutils.BytesToBytes32([]byte("one")),
			val: 1,
			ok:  true,
		}, {
			key: stateutils.BytesToBytes32([]byte("no")),
			val: 0,
			ok:  false,
		},
	}

	m := stateutils.ValidatorIndexMap(state)
	for _, tt := range tests {
		result, ok := m[tt.key]
		if result != tt.val {
			t.Errorf("Expected m[%s] = %d, got %d", tt.key, tt.val, result)
		}
		if ok != tt.ok {
			t.Errorf("Expected ok=%v, got %v", tt.ok, ok)
		}
	}
}
