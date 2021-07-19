package testing

import (
	"math/rand"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/go-bitfield"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	prysmv2 "github.com/prysmaticlabs/prysm/proto/prysm/v2"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/timeutils"
)

// BitlistWithAllBitsSet creates list of bitlists with all bits set.
func BitlistWithAllBitsSet(length uint64) bitfield.Bitlist {
	b := bitfield.NewBitlist(length)
	for i := uint64(0); i < length; i++ {
		b.SetBitAt(i, true)
	}
	return b
}

// BitlistsWithSingleBitSet creates list of bitlists with a single bit set in each.
func BitlistsWithSingleBitSet(n, length uint64) []bitfield.Bitlist {
	lists := make([]bitfield.Bitlist, n)
	for i := uint64(0); i < n; i++ {
		b := bitfield.NewBitlist(length)
		b.SetBitAt(i%length, true)
		lists[i] = b
	}
	return lists
}

// Bitlists64WithSingleBitSet creates list of bitlists with a single bit set in each.
func Bitlists64WithSingleBitSet(n, length uint64) []*bitfield.Bitlist64 {
	lists := make([]*bitfield.Bitlist64, n)
	for i := uint64(0); i < n; i++ {
		b := bitfield.NewBitlist64(length)
		b.SetBitAt(i%length, true)
		lists[i] = b
	}
	return lists
}

// BitlistsWithMultipleBitSet creates list of bitlists with random n bits set.
func BitlistsWithMultipleBitSet(t testing.TB, n, length, count uint64) []bitfield.Bitlist {
	seed := timeutils.Now().UnixNano()
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

// Bitlists64WithMultipleBitSet creates list of bitlists with random n bits set.
func Bitlists64WithMultipleBitSet(t testing.TB, n, length, count uint64) []*bitfield.Bitlist64 {
	seed := timeutils.Now().UnixNano()
	t.Logf("Bitlists64WithMultipleBitSet random seed: %v", seed)
	rand.Seed(seed)
	lists := make([]*bitfield.Bitlist64, n)
	for i := uint64(0); i < n; i++ {
		b := bitfield.NewBitlist64(length)
		keys := rand.Perm(int(length))
		for _, key := range keys[:count] {
			b.SetBitAt(uint64(key), true)
		}
		lists[i] = b
	}
	return lists
}

// MakeAttestationsFromBitlists creates list of attestations from list of bitlist.
func MakeAttestationsFromBitlists(bl []bitfield.Bitlist) []*ethpb.Attestation {
	atts := make([]*ethpb.Attestation, len(bl))
	for i, b := range bl {
		atts[i] = &ethpb.Attestation{
			AggregationBits: b,
			Data: &ethpb.AttestationData{
				Slot:           42,
				CommitteeIndex: 1,
			},
			Signature: bls.NewAggregateSignature().Marshal(),
		}
	}
	return atts
}

// MakeSyncContributionsFromBitVector creates list of sync contributions from list of bitvector.
func MakeSyncContributionsFromBitVector(bl []bitfield.Bitvector128) []*prysmv2.SyncCommitteeContribution {
	c := make([]*prysmv2.SyncCommitteeContribution, len(bl))
	for i, b := range bl {
		c[i] = &prysmv2.SyncCommitteeContribution{
			Slot:              types.Slot(1),
			SubcommitteeIndex: 2,
			AggregationBits:   b,
			Signature:         bls.NewAggregateSignature().Marshal(),
		}
	}
	return c
}
