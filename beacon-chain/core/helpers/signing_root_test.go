package helpers_test

import (
	"bytes"
	"testing"

	fuzz "github.com/google/gofuzz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	ethereum_beacon_p2p_v1 "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	eth "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestSigningRoot_ComputeSigningRoot(t *testing.T) {
	emptyBlock := testutil.NewBeaconBlock()
	_, err := helpers.ComputeSigningRoot(emptyBlock, bytesutil.PadTo([]byte{'T', 'E', 'S', 'T'}, 32))
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
		if got, err := helpers.ComputeDomain(tt.domainType, nil, nil); !bytes.Equal(got, tt.domain) {
			t.Errorf("wanted domain version: %d, got: %d", tt.domain, got)
		} else {
			require.NoError(t, err)
		}
	}
}

func TestSigningRoot_ComputeDomainAndSign(t *testing.T) {
	tests := []struct {
		name       string
		genState   func(t *testing.T) (iface.BeaconState, []bls.SecretKey)
		genBlock   func(t *testing.T, st iface.BeaconState, keys []bls.SecretKey) *eth.SignedBeaconBlock
		domainType [4]byte
		want       []byte
	}{
		{
			name: "block proposer",
			genState: func(t *testing.T) (iface.BeaconState, []bls.SecretKey) {
				beaconState, privKeys := testutil.DeterministicGenesisState(t, 100)
				require.NoError(t, beaconState.SetSlot(beaconState.Slot()+1))
				return beaconState, privKeys
			},
			genBlock: func(t *testing.T, st iface.BeaconState, keys []bls.SecretKey) *eth.SignedBeaconBlock {
				block, err := testutil.GenerateFullBlock(st, keys, nil, 1)
				require.NoError(t, err)
				return block
			},
			domainType: params.BeaconConfig().DomainBeaconProposer,
			want: []byte{
				0x96, 0x65, 0x2a, 0xce, 0x27, 0x68, 0x5b, 0xd8, 0x41, 0xc9, 0xe9, 0x85, 0xd2, 0x33, 0x3, 0xdf, 0x61,
				0xff, 0x2a, 0xb, 0x6c, 0xd, 0x37, 0xad, 0x90, 0xdf, 0xb, 0x3, 0x21, 0xec, 0x23, 0xc5, 0x4e, 0x69, 0x1d,
				0x65, 0x4a, 0xfd, 0x7f, 0xbf, 0x7a, 0xf3, 0xd6, 0xba, 0xfa, 0x57, 0xd7, 0x3c, 0xd, 0x45, 0x21, 0xf4,
				0x78, 0xb9, 0x65, 0x5b, 0x2d, 0x2d, 0xc3, 0x52, 0x1c, 0xef, 0x23, 0x1b, 0xbd, 0x3a, 0x89, 0x6a, 0x5,
				0x1d, 0xaf, 0xaf, 0x0, 0x96, 0xee, 0x72, 0x76, 0xb5, 0xeb, 0x7c, 0xb5, 0xc2, 0xf8, 0xfe, 0x48, 0x6d,
				0x77, 0xbe, 0xd8, 0x4f, 0x55, 0x99, 0x11, 0x1, 0x34, 0xe5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beaconState, privKeys := tt.genState(t)
			idx, err := helpers.BeaconProposerIndex(beaconState)
			require.NoError(t, err)
			block := tt.genBlock(t, beaconState, privKeys)
			got, err := helpers.ComputeDomainAndSign(
				beaconState, helpers.CurrentEpoch(beaconState), block, tt.domainType, privKeys[idx])
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
		digest, err := helpers.ComputeForkDigest(tt.version, tt.root[:])
		require.NoError(t, err)
		assert.Equal(t, tt.result, digest, "Wanted domain version: %#x, got: %#x", digest, tt.result)
	}
}

func TestFuzzverifySigningRoot_10000(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	state := &ethereum_beacon_p2p_v1.BeaconState{}
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
		err := helpers.VerifySigningRoot(state, pubkey[:], sig[:], domain[:])
		_ = err
		err = helpers.VerifySigningRoot(state, p, s, d)
		_ = err
	}
}
