package testing

import (
	"encoding/hex"
	"fmt"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	sszspectest "github.com/prysmaticlabs/go-ssz/spectests"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// Regression tests for investigating discrepancies between ssz signing root of
// our protobuf, simple struct, and python result expected signing root.
// See comments in: https://github.com/prysmaticlabs/prysm/pull/2828
func TestBlockHeaderSigningRoot(t *testing.T) {
	tests := []struct {
		header1      *pb.BeaconBlockHeader
		header2      sszspectest.MainnetBlockHeader
		expectedRoot [32]byte
	}{
		{
			//header = spec.BeaconBlockHeader(
			//	slot = uint64(0),
			//	parent_root = Bytes32(bytes.fromhex('0000000000000000000000000000000000000000000000000000000000000000')),
			//	state_root = Bytes32(bytes.fromhex('03f33c7c997b39605f1fff2b5fa4db1405b193bb9611206cc50afb460960fd6f')),
			//	body_root = Bytes32(bytes.fromhex('0221fd9ca547ba21c5f8df076c7f1b824aeaa208253c63e0ba6c4f6d669d4a5b')),
			//	signature = Bytes96(bytes.fromhex('000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000')),
			//)
			header1: &pb.BeaconBlockHeader{
				StateRoot: hexDecodeOrDie(t, "03f33c7c997b39605f1fff2b5fa4db1405b193bb9611206cc50afb460960fd6f"),
				BodyRoot:  hexDecodeOrDie(t, "0221fd9ca547ba21c5f8df076c7f1b824aeaa208253c63e0ba6c4f6d669d4a5b"),
			},
			header2: sszspectest.MainnetBlockHeader{
				StateRoot: hexDecodeOrDie(t, "03f33c7c997b39605f1fff2b5fa4db1405b193bb9611206cc50afb460960fd6f"),
				BodyRoot:  hexDecodeOrDie(t, "0221fd9ca547ba21c5f8df076c7f1b824aeaa208253c63e0ba6c4f6d669d4a5b"),
			},
			expectedRoot: bytesutil.ToBytes32(hexDecodeOrDie(t, "fa9dfee90cd22268800a48023e7875dd6a67b79fee240b367634fddcc14ed232")),
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			root1, err := ssz.SigningRoot(tt.header1)
			if err != nil {
				t.Error(err)
			}
			root2, err := ssz.SigningRoot(tt.header2)
			if err != nil {
				t.Error(err)
			}

			if root1 != root2 {
				t.Errorf("Root1 = %#x, root2 = %#x. These should be equal!", root1, root2)
			}

			if root1 != tt.expectedRoot {
				t.Errorf("Root1 = %#x, wanted %#x", root1, tt.expectedRoot)
			}
		})
	}
}

func hexDecodeOrDie(t *testing.T, h string) []byte {
	b, err := hex.DecodeString(h)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
