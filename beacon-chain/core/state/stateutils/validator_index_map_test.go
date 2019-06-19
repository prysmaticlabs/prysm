package stateutils_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/stateutils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

func TestValidatorIndexMap_OK(t *testing.T) {
	state := &pb.BeaconState{
		ValidatorRegistry: []*pb.Validator{
			{
				Pubkey: []byte("zero"),
			},
			{
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
			key: bytesutil.ToBytes32([]byte("zero")),
			val: 0,
			ok:  true,
		}, {
			key: bytesutil.ToBytes32([]byte("one")),
			val: 1,
			ok:  true,
		}, {
			key: bytesutil.ToBytes32([]byte("no")),
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
