package tree

//func (c *Checkpoint) GetTree() (*ssz.Node, error) {
//	w := &ssz.Wrapper{}
//	if err := c.GetTreeWithWrapper(w); err != nil {
//		return nil, err
//	}
//	return w.Node(), nil
//}
//
//func (b *BeaconStateAltair) GetTreeWithWrapper(w *ssz.Wrapper) (err error) {
//	indx := w.Indx()
//
//	// Field (0) 'GenesisTime'
//	w.AddUint64(b.GenesisTime)
//
//	// Field (1) 'GenesisValidatorsRoot'
//	if len(b.GenesisValidatorsRoot) != 32 {
//		err = ssz.ErrBytesLength
//		return
//	}
//	w.AddBytes(b.GenesisValidatorsRoot)
//
//	// Field (2) 'Slot'
//	w.AddUint64(uint64(b.Slot))
//
//	// Field (3) 'Fork'
//	if err := b.Fork.GetTreeWithWrapper(w); err != nil {
//		return err
//	}
//
//	// Field (4) 'LatestBlockHeader'
//	if err := b.LatestBlockHeader.GetTreeWithWrapper(w); err != nil {
//		return err
//	}
//
//	// Field (5) 'BlockRoots'
//	{
//		subLeaves := ssz.LeavesFromBytes(b.BlockRoots)
//		tmp, err := ssz.TreeFromNodes(subLeaves)
//		if err != nil {
//			return err
//		}
//		w.AddNode(tmp)
//
//	}
//
//	// Field (6) 'StateRoots'
//	{
//		subLeaves := ssz.LeavesFromBytes(b.StateRoots)
//		tmp, err := ssz.TreeFromNodes(subLeaves)
//		if err != nil {
//			return err
//		}
//		w.AddNode(tmp)
//
//	}
//
//	// Field (7) 'HistoricalRoots'
//	{
//		subLeaves := ssz.LeavesFromBytes(b.HistoricalRoots)
//		numItems := len(b.HistoricalRoots)
//		tmp, err := ssz.TreeFromNodesWithMixin(subLeaves, numItems, int(ssz.CalculateLimit(16777216, uint64(numItems), 8)))
//		if err != nil {
//			return err
//		}
//		w.AddNode(tmp)
//
//	}
//
//	// Field (8) 'Eth1Data'
//	if err := b.Eth1Data.GetTreeWithWrapper(w); err != nil {
//		return err
//	}
//
//	// Field (9) 'Eth1DataVotes'
//	{
//		subIdx := w.Indx()
//		num := len(b.Eth1DataVotes)
//		if num > 2048 {
//			err = ssz.ErrIncorrectListSize
//			return err
//		}
//		for i := 0; i < num; i++ {
//			n, err := b.Eth1DataVotes[i].GetTree()
//			if err != nil {
//				return err
//			}
//			w.AddNode(n)
//		}
//		w.CommitWithMixin(subIdx, num, 2048)
//	}
//
//	// Field (10) 'Eth1DepositIndex'
//	w.AddUint64(b.Eth1DepositIndex)
//
//	// Field (11) 'Validators'
//	{
//		subIdx := w.Indx()
//		num := len(b.Validators)
//		if num > 1099511627776 {
//			err = ssz.ErrIncorrectListSize
//			return err
//		}
//		for i := 0; i < num; i++ {
//			n, err := b.Validators[i].GetTree()
//			if err != nil {
//				return err
//			}
//			w.AddNode(n)
//		}
//		w.CommitWithMixin(subIdx, num, 1099511627776)
//	}
//
//	// Field (12) 'Balances'
//	{
//		subLeaves := ssz.LeavesFromUint64(b.Balances)
//		numItems := len(b.Balances)
//		tmp, err := ssz.TreeFromNodesWithMixin(subLeaves, numItems, int(ssz.CalculateLimit(1099511627776, uint64(numItems), 8)))
//		if err != nil {
//			return err
//		}
//		w.AddNode(tmp)
//
//	}
//
//	// Field (13) 'RandaoMixes'
//	{
//		subLeaves := ssz.LeavesFromBytes(b.RandaoMixes)
//		tmp, err := ssz.TreeFromNodes(subLeaves)
//		if err != nil {
//			return err
//		}
//		w.AddNode(tmp)
//
//	}
//
//	// Field (14) 'Slashings'
//	{
//		subLeaves := ssz.LeavesFromUint64(b.Slashings)
//		tmp, err := ssz.TreeFromNodes(subLeaves)
//		if err != nil {
//			return err
//		}
//		w.AddNode(tmp)
//
//	}
//
//	// Field (15) 'PreviousEpochParticipation'
//	if len(b.PreviousEpochParticipation) > 1099511627776 {
//		err = ssz.ErrBytesLength
//		return
//	}
//	w.AddBytes(b.PreviousEpochParticipation)
//
//	// Field (16) 'CurrentEpochParticipation'
//	if len(b.CurrentEpochParticipation) > 1099511627776 {
//		err = ssz.ErrBytesLength
//		return
//	}
//	w.AddBytes(b.CurrentEpochParticipation)
//
//	// Field (17) 'JustificationBits'
//	if len(b.JustificationBits) != 1 {
//		err = ssz.ErrBytesLength
//		return
//	}
//	w.AddBytes(b.JustificationBits)
//
//	// Field (18) 'PreviousJustifiedCheckpoint'
//	if err := b.PreviousJustifiedCheckpoint.GetTreeWithWrapper(w); err != nil {
//		return err
//	}
//
//	// Field (19) 'CurrentJustifiedCheckpoint'
//	if err := b.CurrentJustifiedCheckpoint.GetTreeWithWrapper(w); err != nil {
//		return err
//	}
//
//	// Field (20) 'FinalizedCheckpoint'
//	if err := b.FinalizedCheckpoint.GetTreeWithWrapper(w); err != nil {
//		return err
//	}
//
//	// Field (21) 'InactivityScores'
//	{
//		subLeaves := ssz.LeavesFromUint64(b.InactivityScores)
//		numItems := len(b.InactivityScores)
//		tmp, err := ssz.TreeFromNodesWithMixin(subLeaves, numItems, int(ssz.CalculateLimit(1099511627776, uint64(numItems), 8)))
//		if err != nil {
//			return err
//		}
//		w.AddNode(tmp)
//
//	}
//
//	// Field (22) 'CurrentSyncCommittee'
//	if err := b.CurrentSyncCommittee.GetTreeWithWrapper(w); err != nil {
//		return err
//	}
//
//	// Field (23) 'NextSyncCommittee'
//	if err := b.NextSyncCommittee.GetTreeWithWrapper(w); err != nil {
//		return err
//	}
//
//	for i := 0; i < 8; i++ {
//		w.AddEmpty()
//	}
//
//	w.Commit(indx)
//	return nil
//}
