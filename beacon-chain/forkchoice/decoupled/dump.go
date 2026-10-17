package decoupled

import (
	"sort"
	"time"
)

// Dump ticks to now, then returns a detached snapshot. Callers serialize access.
func (st *Store) Dump(now time.Time) Dump {
	st.OnTick(now)
	slot := st.slotAt(now)
	fr := st.frame()
	viable := st.viableTree(fr, everyNode)
	confirmed := st.confirmed(now)
	d := Dump{
		Slot:            slot,
		Joined:          st.joined(slot),
		RestartTime:     st.restartTime,
		Head:            st.head(now).Root,
		Confirmed:       confirmed.Root,
		Justified:       st.justified.Root,
		Finalized:       st.finalized.Root,
		Frame:           fr.root.Root,
		JustifiedSlot:   st.justified.Slot,
		FinalizedSlot:   st.finalized.Slot,
		JustifiedHeight: st.hJ,
		FinalizedHeight: st.hF,
		MaxHeight:       st.hMax,
		Floor:           fr.floor,
		CommitteeSize:   st.committeeSize,
	}
	if st.inTree(st.latestConfirmed) {
		d.LatestConfirmed = st.latestConfirmed.Root
	}
	locked := make(map[*Node]bool, len(st.locks))
	for _, l := range st.locks {
		if st.live(l, now) {
			locked[l.block] = true
			d.Locks = append(d.Locks, Lock{Root: l.block.Root, Slot: l.block.Slot, Deadline: l.deadline})
		}
	}
	for _, n := range st.fastConfirmed {
		d.FastConfirmed = append(d.FastConfirmed, n.Root)
	}
	for s, n := range st.confirmations {
		saved := DumpSaved{Slot: s}
		if n != nil {
			saved.Root = n.Root
		}
		d.Saved = append(d.Saved, saved)
	}
	sort.Slice(d.Saved, func(i, j int) bool { return d.Saved[i].Slot < d.Saved[j].Slot })
	for s, b := range st.pool {
		d.Voters = append(d.Voters, DumpVoters{Slot: s, Count: len(b.byValidator)})
	}
	sort.Slice(d.Voters, func(i, j int) bool { return d.Voters[i].Slot < d.Voters[j].Slot })
	var walk func(n *Node)
	walk = func(n *Node) {
		dn := DumpNode{
			Root:      n.Root,
			Slot:      n.Slot,
			FG:        n.FG,
			Votes:     len(n.Votes),
			Support:   len(n.Supported),
			Processed: n.processed,
			Viable:    viable(n),
			Confirmed: isAncestor(n, confirmed),
			Locked:    locked[n],
		}
		if n.Parent != nil {
			dn.Parent = n.Parent.Root
		}
		d.Nodes = append(d.Nodes, dn)
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(st.finalized)
	return d
}
