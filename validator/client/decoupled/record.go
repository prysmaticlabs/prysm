// Package decoupled holds the validator's FG signing record.
package decoupled

import "github.com/OffchainLabs/prysm/v7/consensus-types/primitives"

// Source is the height and entry of the FG source. Zero height means none.
type Source struct {
	Height primitives.Height
	Entry  [32]byte
}

// Head is the head state's justified pair and finalized height.
type Head struct {
	JustifiedHeight primitives.Height
	Justified       [32]byte
	FinalizedHeight primitives.Height
}

// Attestation is an FG vote. Zero FinalityHeight means no finality pair.
type Attestation struct {
	Height         primitives.Height
	Target         [32]byte
	Timeout        bool
	FinalityHeight primitives.Height
	FinalityTarget [32]byte
}

// Record is the validator's FG signing history. It must survive restarts and reorgs.
type Record struct {
	targets  map[primitives.Height][32]byte
	timeouts map[primitives.Height]bool
	locks    map[primitives.Height][32]byte
	named    map[primitives.Height]map[[32]byte]struct{}
}

// NewRecord returns an empty record.
func NewRecord() *Record {
	return &Record{
		targets:  make(map[primitives.Height][32]byte),
		timeouts: make(map[primitives.Height]bool),
		locks:    make(map[primitives.Height][32]byte),
		named:    make(map[primitives.Height]map[[32]byte]struct{}),
	}
}

// Attest returns the FG vote the record allows, false if none. The finality pair is chosen first and locks the height pair. The record is updated before returning.
func (r *Record) Attest(src Source, head Head) (Attestation, bool) {
	a := r.finalityPair(head)
	if !r.heightPair(&a, src) {
		return Attestation{}, false
	}
	// First vote at a height may be a target, each entry is named once per height.
	entries := r.named[a.Height]
	if _, seen := entries[a.Target]; seen || (!a.Timeout && len(entries) > 0) {
		return Attestation{}, false
	}
	if a.FinalityHeight != 0 && a.Height <= a.FinalityHeight {
		a.FinalityHeight, a.FinalityTarget = 0, [32]byte{}
	}
	r.record(a)
	return a, true
}

// Unfinalized justification the record allows.
func (r *Record) finalityPair(head Head) Attestation {
	h := head.JustifiedHeight
	if h <= head.FinalizedHeight || r.timeouts[h] {
		return Attestation{}
	}
	if t, ok := r.targets[h]; ok && t != head.Justified {
		return Attestation{}
	}
	if l, ok := r.locks[h]; ok && l != head.Justified {
		return Attestation{}
	}
	return Attestation{FinalityHeight: h, FinalityTarget: head.Justified}
}

// A lock elsewhere at this height means abstain, a different recorded target means timeout.
func (r *Record) heightPair(a *Attestation, src Source) bool {
	if src.Height == 0 {
		return false
	}
	a.Height, a.Target = src.Height, src.Entry
	if r.timeouts[src.Height] {
		a.Timeout = true
		return true
	}
	lock, locked := r.locks[src.Height]
	if a.FinalityHeight == src.Height {
		lock, locked = a.FinalityTarget, true
	}
	if locked {
		return lock == src.Entry
	}
	if t, ok := r.targets[src.Height]; ok && t != src.Entry {
		a.Timeout = true
	}
	return true
}

// Restore rebuilds Λ from the rows this key signed, in the order they were signed.
func (r *Record) Restore(a Attestation) { r.record(a) }

func (r *Record) record(a Attestation) {
	if r.named[a.Height] == nil {
		r.named[a.Height] = make(map[[32]byte]struct{})
	}
	r.named[a.Height][a.Target] = struct{}{}
	if a.FinalityHeight != 0 {
		r.locks[a.FinalityHeight] = a.FinalityTarget
	}
	if a.Timeout {
		r.timeouts[a.Height] = true
		return
	}
	if _, ok := r.targets[a.Height]; !ok {
		r.targets[a.Height] = a.Target
	}
}
