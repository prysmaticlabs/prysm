package testing

import (
	"math/rand"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
)

// BitlistWithAllBitsSet creates list of bitlists with all bits set.
func BitlistWithAllBitsSet(t testing.TB, length uint64) bitfield.Bitlist {
	b := bitfield.NewBitlist(length)
	for i := uint64(0); i < length; i++ {
		b.SetBitAt(i, true)
	}
	return b
}

// BitlistsWithSingleBitSet creates list of bitlists with a single bit set in each.
func BitlistsWithSingleBitSet(t testing.TB, n, length uint64) []bitfield.Bitlist {
	lists := make([]bitfield.Bitlist, n)
	for i := uint64(0); i < n; i++ {
		b := bitfield.NewBitlist(length)
		b.SetBitAt(i%length, true)
		lists[i] = b
	}
	return lists
}

// BitlistsWithMultipleBitSet creates list of bitlists with random n bits set.
func BitlistsWithMultipleBitSet(t testing.TB, n, length, count uint64) []bitfield.Bitlist {
	seed := roughtime.Now().UnixNano()
	t.Logf("bitlistsWithMultipleBitSet random seed: %v", seed)
	rand.Seed(seed)
	lists := make([]bitfield.Bitlist, n)
	for i := uint64(0); i < n; i++ {
		b := bitfield.NewBitlist(length)
		keys := rand.Perm(int(length))
		for _, key := range keys[:count] {
			b.SetBitAt(uint64(key), true)
		}
		lists[i] = b
	}
	return lists
}

// MakeAttestationsFromBitlists creates list of bitlists from list of attestations.
func MakeAttestationsFromBitlists(t testing.TB, bl []bitfield.Bitlist) []*ethpb.Attestation {
	atts := make([]*ethpb.Attestation, len(bl))
	for i, b := range bl {
		atts[i] = &ethpb.Attestation{
			AggregationBits: b,
			Data:            nil,
			Signature:       bls.NewAggregateSignature().Marshal(),
		}
	}
	return atts
}
