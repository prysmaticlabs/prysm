// Package slasher implements slashing validation
// and proof creation.
package slasher

import (
	"testing"
)

func TestCheckNewProposal_OK(t *testing.T) {
	for ep := uint64(0); ep < 100; ep++ {
		t.Logf("epoch %v", ep)
		for vi := uint64(0); vi < 300000; vi += 10 {
			if !CheckNewProposal(10, ep, vi) {
				t.Fatal("first proposal for epoch by a validator id should always return true")
			}
		}
	}
	for ep := uint64(0); ep < 100; ep++ {
		t.Logf("epoch %v", ep)
		for vi := uint64(1); vi < 300000; vi += 10 {
			if !CheckNewProposal(10, ep, vi) {
				t.Fatal("first proposal for epoch by a validator id should always return true")
			}
		}
	}
	for ep := uint64(0); ep < 100; ep++ {
		t.Logf("epoch %v", ep)
		for vi := uint64(0); vi < 300000; vi += 10 {
			if CheckNewProposal(10, ep, vi) {
				t.Fatal("second proposal for epoch by a validator id should always return false")
			}
		}
	}

}
