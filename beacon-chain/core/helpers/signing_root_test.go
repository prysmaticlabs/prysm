package helpers

import (
	"bytes"
	"testing"

	fuzz "github.com/google/gofuzz"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	ethereum_beacon_p2p_v1 "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestSigningRoot_ComputeOK(t *testing.T) {
	emptyBlock := &ethpb.BeaconBlock{}
	_, err := ComputeSigningRoot(emptyBlock, []byte{'T', 'E', 'S', 'T'})
	assert.NoError(t, err, "Could not compute signing root of block")
}

func TestComputeDomain_OK(t *testing.T) {
	tests := []struct {
		epoch      uint64
		domainType [4]byte
		domain     []byte
	}{
		{epoch: 1, domainType: [4]byte{4, 0, 0, 0}, domain: []byte{4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
		{epoch: 2, domainType: [4]byte{4, 0, 0, 0}, domain: []byte{4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
		{epoch: 2, domainType: [4]byte{5, 0, 0, 0}, domain: []byte{5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
		{epoch: 3, domainType: [4]byte{4, 0, 0, 0}, domain: []byte{4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
		{epoch: 3, domainType: [4]byte{5, 0, 0, 0}, domain: []byte{5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
	}
	for _, tt := range tests {
		if !bytes.Equal(domain(tt.domainType, params.BeaconConfig().ZeroHash[:]), tt.domain) {
			t.Errorf("wanted domain version: %d, got: %d", tt.domain, domain(tt.domainType, params.BeaconConfig().ZeroHash[:]))
		}
	}
}

func TestSigningRoot_Compatibility(t *testing.T) {
	parRoot := [32]byte{'A'}
	stateRoot := [32]byte{'B'}
	blk := &ethpb.BeaconBlock{
		Slot:          20,
		ProposerIndex: 20,
		ParentRoot:    parRoot[:],
		StateRoot:     stateRoot[:],
		Body:          &ethpb.BeaconBlockBody{},
	}
	root, err := ComputeSigningRoot(blk, params.BeaconConfig().DomainBeaconProposer[:])
	require.NoError(t, err)
	newRoot, err := signingData(func() ([32]byte, error) {
		return stateutil.BlockRoot(blk)
	}, params.BeaconConfig().DomainBeaconProposer[:])
	require.NoError(t, err)
	assert.Equal(t, root, newRoot, "Wanted root of %#x but got %#x", root, newRoot)
}

func TestComputeForkDigest_OK(t *testing.T) {
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
		digest, err := ComputeForkDigest(tt.version, tt.root[:])
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
	p := []byte{}
	s := []byte{}
	d := []byte{}
	for i := 0; i < 10000; i++ {
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(&pubkey)
		fuzzer.Fuzz(&sig)
		fuzzer.Fuzz(&domain)
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(&p)
		fuzzer.Fuzz(&s)
		fuzzer.Fuzz(&d)
		err := VerifySigningRoot(state, pubkey[:], sig[:], domain[:])
		err = VerifySigningRoot(state, p, s, d)
		_ = err
	}
}
