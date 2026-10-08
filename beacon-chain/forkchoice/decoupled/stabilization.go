package decoupled

// TODO: stabilization. The walk starts at the FG root until the committee-lock anchor is specified.
func (st *Store) anchor() *Node { return st.fgRoot() }

// TODO: stabilization. The stable record is finality until the committee-lock design lands.
func (st *Store) stableBlock() *Node { return st.finalized }
