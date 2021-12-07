package signing_test

import (
	"bytes"
	"context"
	"testing"

	fuzz "github.com/google/gofuzz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestSigningRoot_ComputeSigningRoot(t *testing.T) {
	emptyBlock := util.NewBeaconBlock()
	_, err := signing.ComputeSigningRoot(emptyBlock, bytesutil.PadTo([]byte{'T', 'E', 'S', 'T'}, 32))
	assert.NoError(t, err, "Could not compute signing root of block")
}

func TestSigningRoot_ComputeDomain(t *testing.T) {
	tests := []struct {
		epoch      uint64
		domainType [4]byte
		domain     []byte
	}{
		{epoch: 1, domainType: [4]byte{4, 0, 0, 0}, domain: []byte{4, 0, 0, 0, 245, 165, 253, 66, 209, 106, 32, 48, 39, 152, 239, 110, 211, 9, 151, 155, 67, 0, 61, 35, 32, 217, 240, 232, 234, 152, 49, 169}},
		{epoch: 2, domainType: [4]byte{4, 0, 0, 0}, domain: []byte{4, 0, 0, 0, 245, 165, 253, 66, 209, 106, 32, 48, 39, 152, 239, 110, 211, 9, 151, 155, 67, 0, 61, 35, 32, 217, 240, 232, 234, 152, 49, 169}},
		{epoch: 2, domainType: [4]byte{5, 0, 0, 0}, domain: []byte{5, 0, 0, 0, 245, 165, 253, 66, 209, 106, 32, 48, 39, 152, 239, 110, 211, 9, 151, 155, 67, 0, 61, 35, 32, 217, 240, 232, 234, 152, 49, 169}},
		{epoch: 3, domainType: [4]byte{4, 0, 0, 0}, domain: []byte{4, 0, 0, 0, 245, 165, 253, 66, 209, 106, 32, 48, 39, 152, 239, 110, 211, 9, 151, 155, 67, 0, 61, 35, 32, 217, 240, 232, 234, 152, 49, 169}},
		{epoch: 3, domainType: [4]byte{5, 0, 0, 0}, domain: []byte{5, 0, 0, 0, 245, 165, 253, 66, 209, 106, 32, 48, 39, 152, 239, 110, 211, 9, 151, 155, 67, 0, 61, 35, 32, 217, 240, 232, 234, 152, 49, 169}},
	}
	for _, tt := range tests {
		if got, err := signing.ComputeDomain(tt.domainType, nil, nil); !bytes.Equal(got, tt.domain) {
			t.Errorf("wanted domain version: %d, got: %d", tt.domain, got)
		} else {
			require.NoError(t, err)
		}
	}
}

func TestSigningRoot_ComputeDomainAndSign(t *testing.T) {
	tests := []struct {
		name       string
		genState   func(t *testing.T) (state.BeaconState, []bls.SecretKey)
		genBlock   func(t *testing.T, st state.BeaconState, keys []bls.SecretKey) *ethpb.SignedBeaconBlock
		domainType [4]byte
		want       []byte
	}{
		{
			name: "block proposer",
			genState: func(t *testing.T) (state.BeaconState, []bls.SecretKey) {
				beaconState, privKeys := util.DeterministicGenesisState(t, 100)
				require.NoError(t, beaconState.SetSlot(beaconState.Slot()+1))
				return beaconState, privKeys
			},
			genBlock: func(t *testing.T, st state.BeaconState, keys []bls.SecretKey) *ethpb.SignedBeaconBlock {
				block, err := util.GenerateFullBlock(st, keys, nil, 1)
				require.NoError(t, err)
				return block
			},
			domainType: params.BeaconConfig().DomainBeaconProposer,
			want: []byte{
				0xad, 0xd8, 0xf0, 0xd1, 0xae, 0x82, 0xaa, 0x3, 0x9a, 0xcd, 0x8e, 0xb7, 0x84, 0x14, 0x1c, 0x21, 0x81,
				0xbc, 0x1b, 0x2, 0xb5, 0x6d, 0x4c, 0x76, 0x36, 0x5f, 0xba, 0x6e, 0x33, 0x9e, 0xda, 0xe, 0x36, 0xe1,
				0xf, 0x30, 0xae, 0x6, 0x44, 0xd4, 0x38, 0x21, 0xf0, 0x45, 0xc2, 0x54, 0x68, 0x2f, 0x12, 0xcc, 0x27,
				0x45, 0x72, 0x5, 0xaf, 0xb4, 0x85, 0x60, 0xdb, 0x7a, 0x1f, 0xe7, 0xa8, 0x62, 0xf5, 0x71, 0xac, 0x88,
				0x8c, 0xd3, 0xba, 0x4d, 0xa3, 0x3d, 0x3b, 0x87, 0x9b, 0x23, 0xae, 0xe4, 0x46, 0xc6, 0x36, 0xca, 0xa5,
				0xa1, 0x2d, 0x9e, 0x7, 0xc1, 0x40, 0xed, 0x99, 0xfd, 0xae, 0xce,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beaconState, privKeys := tt.genState(t)
			idx, err := helpers.BeaconProposerIndex(context.Background(), beaconState)
			require.NoError(t, err)
			block := tt.genBlock(t, beaconState, privKeys)
			got, err := signing.ComputeDomainAndSign(
				beaconState, time.CurrentEpoch(beaconState), block, tt.domainType, privKeys[idx])
			require.NoError(t, err)
			require.DeepEqual(t, tt.want, got, "Incorrect signature")
		})
	}
}

func TestSigningRoot_ComputeForkDigest(t *testing.T) {
	tests := []struct {
		version []byte
		root    [32]byte
		result  [4]byte
	}{
		{version: []byte{'A', 'B', 'C', 'D'}, root: [32]byte{'i', 'o', 'p'}, result: [4]byte{0x69, 0x5c, 0x26, 0x47}},
		{version: []byte{'i', 'm', 'n', 'a'}, root: [32]byte{'z', 'a', 'b'}, result: [4]byte{0x1c, 0x38, 0x84, 0x58}},
		{version: []byte{'b', 'w', 'r', 't'}, root: [32]byte{'r', 'd', 'c'}, result: [4]byte{0x83, 0x34, 0x38, 0x88}},
	}
	for _, tt := range tests {
		digest, err := signing.ComputeForkDigest(tt.version, tt.root[:])
		require.NoError(t, err)
		assert.Equal(t, tt.result, digest, "Wanted domain version: %#x, got: %#x", digest, tt.result)
	}
}

func TestFuzzverifySigningRoot_10000(_ *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	state := &ethpb.BeaconState{}
	pubkey := [48]byte{}
	sig := [96]byte{}
	domain := [4]byte{}
	var p []byte
	var s []byte
	var d []byte
	for i := 0; i < 10000; i++ {
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(&pubkey)
		fuzzer.Fuzz(&sig)
		fuzzer.Fuzz(&domain)
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(&p)
		fuzzer.Fuzz(&s)
		fuzzer.Fuzz(&d)
		err := signing.VerifySigningRoot(state, pubkey[:], sig[:], domain[:])
		_ = err
		err = signing.VerifySigningRoot(state, p, s, d)
		_ = err
	}
}
