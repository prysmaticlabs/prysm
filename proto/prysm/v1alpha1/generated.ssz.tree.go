package eth

import (
	ssz "github.com/ferranbt/fastssz"
)

// GetTree returns tree-backing for the Checkpoint object
func (c *Checkpoint) GetTreeWithWrapper(w *ssz.Wrapper) (err error) {
	indx := w.Indx()

	// Field (0) 'Epoch'
	w.AddUint64(uint64(c.Epoch))

	// Field (1) 'Root'
	if len(c.Root) != 32 {
		err = ssz.ErrBytesLength
		return
	}
	w.AddBytes(c.Root)

	w.Commit(indx)
	return nil
}

func (c *Checkpoint) GetTree() (*ssz.Node, error) {
	w := &ssz.Wrapper{}
	if err := c.GetTreeWithWrapper(w); err != nil {
		return nil, err
	}
	return w.Node(), nil
}

// GetTree returns tree-backing for the Eth1Data object
func (e *Eth1Data) GetTreeWithWrapper(w *ssz.Wrapper) (err error) {
	indx := w.Indx()

	// Field (0) 'DepositRoot'
	if len(e.DepositRoot) != 32 {
		err = ssz.ErrBytesLength
		return
	}
	w.AddBytes(e.DepositRoot)

	// Field (1) 'DepositCount'
	w.AddUint64(e.DepositCount)

	// Field (2) 'BlockHash'
	if len(e.BlockHash) != 32 {
		err = ssz.ErrBytesLength
		return
	}
	w.AddBytes(e.BlockHash)

	for i := 0; i < 1; i++ {
		w.AddEmpty()
	}

	w.Commit(indx)
	return nil
}

func (e *Eth1Data) GetTree() (*ssz.Node, error) {
	w := &ssz.Wrapper{}
	if err := e.GetTreeWithWrapper(w); err != nil {
		return nil, err
	}
	return w.Node(), nil
}

// GetTree returns tree-backing for the BeaconBlockHeader object
func (b *BeaconBlockHeader) GetTreeWithWrapper(w *ssz.Wrapper) (err error) {
	indx := w.Indx()

	// Field (0) 'Slot'
	w.AddUint64(uint64(b.Slot))

	// Field (1) 'ProposerIndex'
	w.AddUint64(uint64(b.ProposerIndex))

	// Field (2) 'ParentRoot'
	if len(b.ParentRoot) != 32 {
		err = ssz.ErrBytesLength
		return
	}
	w.AddBytes(b.ParentRoot)

	// Field (3) 'StateRoot'
	if len(b.StateRoot) != 32 {
		err = ssz.ErrBytesLength
		return
	}
	w.AddBytes(b.StateRoot)

	// Field (4) 'BodyRoot'
	if len(b.BodyRoot) != 32 {
		err = ssz.ErrBytesLength
		return
	}
	w.AddBytes(b.BodyRoot)

	for i := 0; i < 3; i++ {
		w.AddEmpty()
	}

	w.Commit(indx)
	return nil
}

func (b *BeaconBlockHeader) GetTree() (*ssz.Node, error) {
	w := &ssz.Wrapper{}
	if err := b.GetTreeWithWrapper(w); err != nil {
		return nil, err
	}
	return w.Node(), nil
}

// GetTree returns tree-backing for the SyncAggregate object
func (s *SyncAggregate) GetTreeWithWrapper(w *ssz.Wrapper) (err error) {
	indx := w.Indx()

	// Field (0) 'SyncCommitteeBits'
	if len(s.SyncCommitteeBits) != 64 {
		err = ssz.ErrBytesLength
		return
	}
	w.AddBytes(s.SyncCommitteeBits)

	// Field (1) 'SyncCommitteeSignature'
	if len(s.SyncCommitteeSignature) != 96 {
		err = ssz.ErrBytesLength
		return
	}
	w.AddBytes(s.SyncCommitteeSignature)

	w.Commit(indx)
	return nil
}

func (s *SyncAggregate) GetTree() (*ssz.Node, error) {
	w := &ssz.Wrapper{}
	if err := s.GetTreeWithWrapper(w); err != nil {
		return nil, err
	}
	return w.Node(), nil
}

// GetTree returns tree-backing for the BeaconStateAltair object
func (b *BeaconStateAltair) GetTreeWithWrapper(w *ssz.Wrapper) (err error) {
	indx := w.Indx()

	// Field (0) 'GenesisTime'
	w.AddUint64(b.GenesisTime)

	// Field (1) 'GenesisValidatorsRoot'
	if len(b.GenesisValidatorsRoot) != 32 {
		err = ssz.ErrBytesLength
		return
	}
	w.AddBytes(b.GenesisValidatorsRoot)

	// Field (2) 'Slot'
	w.AddUint64(uint64(b.Slot))

	// Field (3) 'Fork'
	if err := b.Fork.GetTreeWithWrapper(w); err != nil {
		return err
	}

	// Field (4) 'LatestBlockHeader'
	if err := b.LatestBlockHeader.GetTreeWithWrapper(w); err != nil {
		return err
	}

	// Field (5) 'BlockRoots'
	{
		subLeaves := ssz.LeavesFromBytes(b.BlockRoots)
		tmp, err := ssz.TreeFromNodes(subLeaves)
		if err != nil {
			return err
		}
		w.AddNode(tmp)

	}

	// Field (6) 'StateRoots'
	{
		subLeaves := ssz.LeavesFromBytes(b.StateRoots)
		tmp, err := ssz.TreeFromNodes(subLeaves)
		if err != nil {
			return err
		}
		w.AddNode(tmp)

	}

	// Field (7) 'HistoricalRoots'
	{
		subLeaves := ssz.LeavesFromBytes(b.HistoricalRoots)
		numItems := len(b.HistoricalRoots)
		tmp, err := ssz.TreeFromNodesWithMixin(subLeaves, numItems, int(ssz.CalculateLimit(16777216, uint64(numItems), 8)))
		if err != nil {
			return err
		}
		w.AddNode(tmp)

	}

	// Field (8) 'Eth1Data'
	if err := b.Eth1Data.GetTreeWithWrapper(w); err != nil {
		return err
	}

	// Field (9) 'Eth1DataVotes'
	{
		subIdx := w.Indx()
		num := len(b.Eth1DataVotes)
		if num > 2048 {
			err = ssz.ErrIncorrectListSize
			return err
		}
		for i := 0; i < num; i++ {
			n, err := b.Eth1DataVotes[i].GetTree()
			if err != nil {
				return err
			}
			w.AddNode(n)
		}
		w.CommitWithMixin(subIdx, num, 2048)
	}

	// Field (10) 'Eth1DepositIndex'
	w.AddUint64(b.Eth1DepositIndex)

	// Field (11) 'Validators'
	{
		subIdx := w.Indx()
		num := len(b.Validators)
		if num > 1099511627776 {
			err = ssz.ErrIncorrectListSize
			return err
		}
		for i := 0; i < num; i++ {
			n, err := b.Validators[i].GetTree()
			if err != nil {
				return err
			}
			w.AddNode(n)
		}
		w.CommitWithMixin(subIdx, num, 1099511627776)
	}

	// Field (12) 'Balances'
	{
		subLeaves := ssz.LeavesFromUint64(b.Balances)
		numItems := len(b.Balances)
		tmp, err := ssz.TreeFromNodesWithMixin(subLeaves, numItems, int(ssz.CalculateLimit(1099511627776, uint64(numItems), 8)))
		if err != nil {
			return err
		}
		w.AddNode(tmp)

	}

	// Field (13) 'RandaoMixes'
	{
		subLeaves := ssz.LeavesFromBytes(b.RandaoMixes)
		tmp, err := ssz.TreeFromNodes(subLeaves)
		if err != nil {
			return err
		}
		w.AddNode(tmp)

	}

	// Field (14) 'Slashings'
	{
		subLeaves := ssz.LeavesFromUint64(b.Slashings)
		tmp, err := ssz.TreeFromNodes(subLeaves)
		if err != nil {
			return err
		}
		w.AddNode(tmp)

	}

	// Field (15) 'PreviousEpochParticipation'
	if len(b.PreviousEpochParticipation) > 1099511627776 {
		err = ssz.ErrBytesLength
		return
	}
	w.AddBytes(b.PreviousEpochParticipation)

	// Field (16) 'CurrentEpochParticipation'
	if len(b.CurrentEpochParticipation) > 1099511627776 {
		err = ssz.ErrBytesLength
		return
	}
	w.AddBytes(b.CurrentEpochParticipation)

	// Field (17) 'JustificationBits'
	if len(b.JustificationBits) != 1 {
		err = ssz.ErrBytesLength
		return
	}
	w.AddBytes(b.JustificationBits)

	// Field (18) 'PreviousJustifiedCheckpoint'
	if err := b.PreviousJustifiedCheckpoint.GetTreeWithWrapper(w); err != nil {
		return err
	}

	// Field (19) 'CurrentJustifiedCheckpoint'
	if err := b.CurrentJustifiedCheckpoint.GetTreeWithWrapper(w); err != nil {
		return err
	}

	// Field (20) 'FinalizedCheckpoint'
	if err := b.FinalizedCheckpoint.GetTreeWithWrapper(w); err != nil {
		return err
	}

	// Field (21) 'InactivityScores'
	{
		subLeaves := ssz.LeavesFromUint64(b.InactivityScores)
		numItems := len(b.InactivityScores)
		tmp, err := ssz.TreeFromNodesWithMixin(subLeaves, numItems, int(ssz.CalculateLimit(1099511627776, uint64(numItems), 8)))
		if err != nil {
			return err
		}
		w.AddNode(tmp)

	}

	// Field (22) 'CurrentSyncCommittee'
	if err := b.CurrentSyncCommittee.GetTreeWithWrapper(w); err != nil {
		return err
	}

	// Field (23) 'NextSyncCommittee'
	if err := b.NextSyncCommittee.GetTreeWithWrapper(w); err != nil {
		return err
	}

	for i := 0; i < 8; i++ {
		w.AddEmpty()
	}

	w.Commit(indx)
	return nil
}

func (b *BeaconStateAltair) GetTree() (*ssz.Node, error) {
	w := &ssz.Wrapper{}
	if err := b.GetTreeWithWrapper(w); err != nil {
		return nil, err
	}
	return w.Node(), nil
}

// GetTree returns tree-backing for the Fork object
func (f *Fork) GetTreeWithWrapper(w *ssz.Wrapper) (err error) {
	indx := w.Indx()

	// Field (0) 'PreviousVersion'
	if len(f.PreviousVersion) != 4 {
		err = ssz.ErrBytesLength
		return
	}
	w.AddBytes(f.PreviousVersion)

	// Field (1) 'CurrentVersion'
	if len(f.CurrentVersion) != 4 {
		err = ssz.ErrBytesLength
		return
	}
	w.AddBytes(f.CurrentVersion)

	// Field (2) 'Epoch'
	w.AddUint64(uint64(f.Epoch))

	for i := 0; i < 1; i++ {
		w.AddEmpty()
	}

	w.Commit(indx)
	return nil
}

func (f *Fork) GetTree() (*ssz.Node, error) {
	w := &ssz.Wrapper{}
	if err := f.GetTreeWithWrapper(w); err != nil {
		return nil, err
	}
	return w.Node(), nil
}

// GetTree returns tree-backing for the ForkData object
func (f *ForkData) GetTreeWithWrapper(w *ssz.Wrapper) (err error) {
	indx := w.Indx()

	// Field (0) 'CurrentVersion'
	if len(f.CurrentVersion) != 4 {
		err = ssz.ErrBytesLength
		return
	}
	w.AddBytes(f.CurrentVersion)

	// Field (1) 'GenesisValidatorsRoot'
	if len(f.GenesisValidatorsRoot) != 32 {
		err = ssz.ErrBytesLength
		return
	}
	w.AddBytes(f.GenesisValidatorsRoot)

	w.Commit(indx)
	return nil
}

func (f *ForkData) GetTree() (*ssz.Node, error) {
	w := &ssz.Wrapper{}
	if err := f.GetTreeWithWrapper(w); err != nil {
		return nil, err
	}
	return w.Node(), nil
}

// GetTree returns tree-backing for the SyncCommittee object
func (s *SyncCommittee) GetTreeWithWrapper(w *ssz.Wrapper) (err error) {
	indx := w.Indx()

	// Field (0) 'Pubkeys'
	{
		subLeaves := ssz.LeavesFromBytes(s.Pubkeys)
		tmp, err := ssz.TreeFromNodes(subLeaves)
		if err != nil {
			return err
		}
		w.AddNode(tmp)

	}

	// Field (1) 'AggregatePubkey'
	if len(s.AggregatePubkey) != 48 {
		err = ssz.ErrBytesLength
		return
	}
	w.AddBytes(s.AggregatePubkey)

	w.Commit(indx)
	return nil
}

func (s *SyncCommittee) GetTree() (*ssz.Node, error) {
	w := &ssz.Wrapper{}
	if err := s.GetTreeWithWrapper(w); err != nil {
		return nil, err
	}
	return w.Node(), nil
}

// GetTree returns tree-backing for the SyncCommitteeMessage object
func (s *SyncCommitteeMessage) GetTreeWithWrapper(w *ssz.Wrapper) (err error) {
	indx := w.Indx()

	// Field (0) 'Slot'
	w.AddUint64(uint64(s.Slot))

	// Field (1) 'BlockRoot'
	if len(s.BlockRoot) != 32 {
		err = ssz.ErrBytesLength
		return
	}
	w.AddBytes(s.BlockRoot)

	// Field (2) 'ValidatorIndex'
	w.AddUint64(uint64(s.ValidatorIndex))

	// Field (3) 'Signature'
	if len(s.Signature) != 96 {
		err = ssz.ErrBytesLength
		return
	}
	w.AddBytes(s.Signature)

	w.Commit(indx)
	return nil
}

func (s *SyncCommitteeMessage) GetTree() (*ssz.Node, error) {
	w := &ssz.Wrapper{}
	if err := s.GetTreeWithWrapper(w); err != nil {
		return nil, err
	}
	return w.Node(), nil
}

// GetTree returns tree-backing for the SyncCommitteeContribution object
func (s *SyncCommitteeContribution) GetTreeWithWrapper(w *ssz.Wrapper) (err error) {
	indx := w.Indx()

	// Field (0) 'Slot'
	w.AddUint64(uint64(s.Slot))

	// Field (1) 'BlockRoot'
	if len(s.BlockRoot) != 32 {
		err = ssz.ErrBytesLength
		return
	}
	w.AddBytes(s.BlockRoot)

	// Field (2) 'SubcommitteeIndex'
	w.AddUint64(s.SubcommitteeIndex)

	// Field (3) 'AggregationBits'
	if len(s.AggregationBits) != 16 {
		err = ssz.ErrBytesLength
		return
	}
	w.AddBytes(s.AggregationBits)

	// Field (4) 'Signature'
	if len(s.Signature) != 96 {
		err = ssz.ErrBytesLength
		return
	}
	w.AddBytes(s.Signature)

	for i := 0; i < 3; i++ {
		w.AddEmpty()
	}

	w.Commit(indx)
	return nil
}

func (s *SyncCommitteeContribution) GetTree() (*ssz.Node, error) {
	w := &ssz.Wrapper{}
	if err := s.GetTreeWithWrapper(w); err != nil {
		return nil, err
	}
	return w.Node(), nil
}

// GetTree returns tree-backing for the ContributionAndProof object
func (c *ContributionAndProof) GetTreeWithWrapper(w *ssz.Wrapper) (err error) {
	indx := w.Indx()

	// Field (0) 'AggregatorIndex'
	w.AddUint64(uint64(c.AggregatorIndex))

	// Field (1) 'Contribution'
	if err := c.Contribution.GetTreeWithWrapper(w); err != nil {
		return err
	}

	// Field (2) 'SelectionProof'
	if len(c.SelectionProof) != 96 {
		err = ssz.ErrBytesLength
		return
	}
	w.AddBytes(c.SelectionProof)

	for i := 0; i < 1; i++ {
		w.AddEmpty()
	}

	w.Commit(indx)
	return nil
}

func (c *ContributionAndProof) GetTree() (*ssz.Node, error) {
	w := &ssz.Wrapper{}
	if err := c.GetTreeWithWrapper(w); err != nil {
		return nil, err
	}
	return w.Node(), nil
}

// GetTree returns tree-backing for the SignedContributionAndProof object
func (s *SignedContributionAndProof) GetTreeWithWrapper(w *ssz.Wrapper) (err error) {
	indx := w.Indx()

	// Field (0) 'Message'
	if err := s.Message.GetTreeWithWrapper(w); err != nil {
		return err
	}

	// Field (1) 'Signature'
	if len(s.Signature) != 96 {
		err = ssz.ErrBytesLength
		return
	}
	w.AddBytes(s.Signature)

	w.Commit(indx)
	return nil
}

func (s *SignedContributionAndProof) GetTree() (*ssz.Node, error) {
	w := &ssz.Wrapper{}
	if err := s.GetTreeWithWrapper(w); err != nil {
		return nil, err
	}
	return w.Node(), nil
}

// GetTree returns tree-backing for the Validator object
func (v *Validator) GetTreeWithWrapper(w *ssz.Wrapper) (err error) {
	indx := w.Indx()

	// Field (0) 'PublicKey'
	if len(v.PublicKey) != 48 {
		err = ssz.ErrBytesLength
		return
	}
	buf := make([]byte, 48)
	copy(buf, v.PublicKey)
	zeroBytes := make([]byte, 32)
	if rest := len(buf) % 32; rest != 0 {
		// pad zero bytes to the left
		buf = append(buf, zeroBytes[:32-rest]...)
	}
	items := make([][]byte, 2)
	items[0] = zeroBytes
	items[1] = zeroBytes
	copy(items[0], buf[0:32])
	copy(items[1], buf[32:64])
	var leaf *ssz.Node
	leaf, err = ssz.TreeFromChunks(items)
	if err != nil {
		return
	}
	w.AddNode(leaf)

	// Field (1) 'WithdrawalCredentials'
	if len(v.WithdrawalCredentials) != 32 {
		err = ssz.ErrBytesLength
		return
	}
	w.AddBytes(v.WithdrawalCredentials)

	// Field (2) 'EffectiveBalance'
	w.AddUint64(v.EffectiveBalance)

	// Field (3) 'Slashed'
	tmp := ssz.LeafFromBool(v.Slashed)
	w.AddNode(tmp)

	// Field (4) 'ActivationEligibilityEpoch'
	w.AddUint64(uint64(v.ActivationEligibilityEpoch))

	// Field (5) 'ActivationEpoch'
	w.AddUint64(uint64(v.ActivationEpoch))

	// Field (6) 'ExitEpoch'
	w.AddUint64(uint64(v.ExitEpoch))

	// Field (7) 'WithdrawableEpoch'
	w.AddUint64(uint64(v.WithdrawableEpoch))

	w.Commit(indx)
	return nil
}

func (v *Validator) GetTree() (*ssz.Node, error) {
	w := &ssz.Wrapper{}
	if err := v.GetTreeWithWrapper(w); err != nil {
		return nil, err
	}
	return w.Node(), nil
}
