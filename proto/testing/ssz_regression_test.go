package testing

import (
	"encoding/hex"
	"fmt"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	sszspectest "github.com/prysmaticlabs/go-ssz/spectests"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

// Regression tests for investigating discrepancies between ssz signing root of
// our protobuf, simple struct, and python result expected signing root.
// See comments in: https://github.com/prysmaticlabs/prysm/pull/2828
func TestBlockHeaderSigningRoot(t *testing.T) {
	t.Skip("Needs updated data after v0.9.3 rm signing root PR")
	tests := []struct {
		header1      *ethpb.BeaconBlockHeader
		header2      sszspectest.MainnetBlockHeader
		expectedRoot [32]byte
	}{
		{
			// Arbitrary example, validated by running in python.
			//header = spec.BeaconBlockHeader(
			//	slot = uint64(0),
			//	parent_root = Bytes32(bytes.fromhex('0000000000000000000000000000000000000000000000000000000000000000')),
			//	state_root = Bytes32(bytes.fromhex('03f33c7c997b39605f1fff2b5fa4db1405b193bb9611206cc50afb460960fd6f')),
			//	body_root = Bytes32(bytes.fromhex('0221fd9ca547ba21c5f8df076c7f1b824aeaa208253c63e0ba6c4f6d669d4a5b')),
			//	signature = Bytes96(bytes.fromhex('000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000')),
			//)
			header1: &ethpb.BeaconBlockHeader{
				StateRoot: hexDecodeOrDie(t, "03f33c7c997b39605f1fff2b5fa4db1405b193bb9611206cc50afb460960fd6f"),
				BodyRoot:  hexDecodeOrDie(t, "0221fd9ca547ba21c5f8df076c7f1b824aeaa208253c63e0ba6c4f6d669d4a5b"),
			},
			header2: sszspectest.MainnetBlockHeader{
				StateRoot: hexDecodeOrDie(t, "03f33c7c997b39605f1fff2b5fa4db1405b193bb9611206cc50afb460960fd6f"),
				BodyRoot:  hexDecodeOrDie(t, "0221fd9ca547ba21c5f8df076c7f1b824aeaa208253c63e0ba6c4f6d669d4a5b"),
			},
			expectedRoot: bytesutil.ToBytes32(hexDecodeOrDie(t, "fa9dfee90cd22268800a48023e7875dd6a67b79fee240b367634fddcc14ed232")),
		},
		{
			// First example from 0.8 ssz_mainnet_random.yaml.
			//    value: {slot: 14215038047959786547, parent_root: '0xf9b2785de53069d4ad16cc0ec729afe9f879e391433ec120bb15b5082a486705',
			//      state_root: '0x737d1c6ff6e2edf7f0627bf55381e6b08f6c2c56ed8d1895ae47a782dc09382e',
			//      body_root: '0xaffff5006c34a3a2bf7f18b7860675f002187ea809f708fa8f44c424321bcd1c',
			//      signature: '0x17d25044259a0ccd99d1b45eeec4e084e5fb0fef98d5805001b248feb555b947ecf6842b9ad546f98f63ef89117575d73223e9fb9ee8143857b6fcc79600fffed1966cea46f7524236cd1e83531aef906cb8b4c296d50695bb83efa84075d309'}
			//    serialized: '0x33244e45caf545c5f9b2785de53069d4ad16cc0ec729afe9f879e391433ec120bb15b5082a486705737d1c6ff6e2edf7f0627bf55381e6b08f6c2c56ed8d1895ae47a782dc09382eaffff5006c34a3a2bf7f18b7860675f002187ea809f708fa8f44c424321bcd1c17d25044259a0ccd99d1b45eeec4e084e5fb0fef98d5805001b248feb555b947ecf6842b9ad546f98f63ef89117575d73223e9fb9ee8143857b6fcc79600fffed1966cea46f7524236cd1e83531aef906cb8b4c296d50695bb83efa84075d309'
			//    root: '0x6ae0bafe59ff0bab856c3f26c392dfca9c32d395b0ceccdddf0bee95120facd9'
			//    signing_root: '0xa7b0199ee4cd6b9d764ca93ee285fb98313ddd4994c52b5d64dd75a3c4b2b85a'
			header1: &ethpb.BeaconBlockHeader{
				Slot:       14215038047959786547,
				ParentRoot: hexDecodeOrDie(t, "f9b2785de53069d4ad16cc0ec729afe9f879e391433ec120bb15b5082a486705"),
				StateRoot:  hexDecodeOrDie(t, "737d1c6ff6e2edf7f0627bf55381e6b08f6c2c56ed8d1895ae47a782dc09382e"),
				BodyRoot:   hexDecodeOrDie(t, "affff5006c34a3a2bf7f18b7860675f002187ea809f708fa8f44c424321bcd1c"),
			},
			header2: sszspectest.MainnetBlockHeader{
				Slot:       14215038047959786547,
				ParentRoot: hexDecodeOrDie(t, "f9b2785de53069d4ad16cc0ec729afe9f879e391433ec120bb15b5082a486705"),
				StateRoot:  hexDecodeOrDie(t, "737d1c6ff6e2edf7f0627bf55381e6b08f6c2c56ed8d1895ae47a782dc09382e"),
				BodyRoot:   hexDecodeOrDie(t, "affff5006c34a3a2bf7f18b7860675f002187ea809f708fa8f44c424321bcd1c"),
				Signature:  hexDecodeOrDie(t, "17d25044259a0ccd99d1b45eeec4e084e5fb0fef98d5805001b248feb555b947ecf6842b9ad546f98f63ef89117575d73223e9fb9ee8143857b6fcc79600fffed1966cea46f7524236cd1e83531aef906cb8b4c296d50695bb83efa84075d309"),
			},
			expectedRoot: bytesutil.ToBytes32(hexDecodeOrDie(t, "a7b0199ee4cd6b9d764ca93ee285fb98313ddd4994c52b5d64dd75a3c4b2b85a")),
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			root1, err := tt.header1.HashTreeRoot()
			if err != nil {
				t.Error(err)
			}
			root2, err := ssz.HashTreeRoot(tt.header2)
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
