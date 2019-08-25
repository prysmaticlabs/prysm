// Package slasher implements slashing validation
// and proof creation.
package slasher

import (
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
)

func TestCheckNewProposal_PopulateAndValidate(t *testing.T) {
	epochProposalBitlist = make(map[uint64]bitfield.Bitlist)

	for ep := uint64(0); ep < 100; ep++ {
		for vi := uint64(0); vi < 300000; vi += 10 {
			if first, err := CheckNewProposal(10, ep, vi); err != nil || !first {
				t.Fatal("first proposal for epoch by a validator id should always return true")
			}
		}
	}

	for ep := uint64(0); ep < 100; ep++ {
		for vi := uint64(1); vi < 300000; vi += 10 {
			if first, err := CheckNewProposal(10, ep, vi); err != nil || !first {
				t.Fatal("first proposal for epoch by a validator id should always return true")
			}
		}
	}

	for ep := uint64(0); ep < 100; ep++ {
		for vi := uint64(0); vi < 300000; vi += 10 {
			if first, err := CheckNewProposal(10, ep, vi); err != nil || first {
				t.Fatal("second proposal for epoch by a validator id should always return false")
			}
		}
	}
}

func TestCheckNewProposal_ErrorOnOldProposals(t *testing.T) {
	epochProposalBitlist = make(map[uint64]bitfield.Bitlist)

	for ep := uint64(0); ep < 100; ep++ {
		for vi := uint64(0); vi < 300000; vi += 10 {
			if first, err := CheckNewProposal(10, ep, vi); err != nil || !first {
				t.Fatal("first proposal for epoch by a validator id should always return true")
			}
		}
	}

	for ep := uint64(0); ep < 100; ep++ {
		for vi := uint64(0); vi < 300000; vi += 10 {
			if _, err := CheckNewProposal(weakSubjectivityPeriod+ep+1, ep, vi); err == nil {
				t.Fatal("proposals older then weak subjectivity period should return error")
			}
		}
	}
}

func TestCheckNewProposal_TruncateOldProposals(t *testing.T) {
	epochProposalBitlist = make(map[uint64]bitfield.Bitlist)

	for ep := uint64(0); ep < 100; ep++ {
		for vi := uint64(0); vi < 300000; vi += 10 {
			if first, err := CheckNewProposal(10, ep, vi); err != nil || !first {
				t.Fatal("first proposal for epoch by a validator id should always return true")
			}
		}
	}

	for ep := uint64(weakSubjectivityPeriod + 1); ep < weakSubjectivityPeriod+100; ep++ {
		for vi := uint64(0); vi < 300000; vi += 10 {
			if first, err := CheckNewProposal(ep, ep, vi); err != nil || !first {
				t.Fatal("first proposal for epoch by a validator id should always return true")
			}
		}
	}

	for ep := uint64(0); ep < 100; ep++ {
		_, ok := epochProposalBitlist[ep]
		if ok {
			t.Fatal("proposals older then weak subjectivity period should have been truncated")
		}
	}
}

func BenchmarkTimeToPopulate(b *testing.B) {
	for ep := uint64(0); ep < uint64(b.N); ep++ {
		for vi := uint64(0); vi < 300000; vi++ {
			_, err := CheckNewProposal(10, ep, vi)
			if err != nil {
				b.Error(err)
			}
		}
	}
}
