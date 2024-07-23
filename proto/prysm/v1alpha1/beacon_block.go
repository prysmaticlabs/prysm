package eth

import "github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"

// Copy --
func (data *Eth1Data) Copy() *Eth1Data {
	if data == nil {
		return nil
	}
	return &Eth1Data{
		DepositRoot:  bytesutil.SafeCopyBytes(data.DepositRoot),
		DepositCount: data.DepositCount,
		BlockHash:    bytesutil.SafeCopyBytes(data.BlockHash),
	}
}

// Copy --
func (slashing *ProposerSlashing) Copy() *ProposerSlashing {
	if slashing == nil {
		return nil
	}
	return &ProposerSlashing{
		Header_1: slashing.Header_1.Copy(),
		Header_2: slashing.Header_2.Copy(),
	}
}

// Copy --
func (header *SignedBeaconBlockHeader) Copy() *SignedBeaconBlockHeader {
	if header == nil {
		return nil
	}
	return &SignedBeaconBlockHeader{
		Header:    header.Header.Copy(),
		Signature: bytesutil.SafeCopyBytes(header.Signature),
	}
}

// Copy --
func (header *BeaconBlockHeader) Copy() *BeaconBlockHeader {
	if header == nil {
		return nil
	}
	parentRoot := bytesutil.SafeCopyBytes(header.ParentRoot)
	stateRoot := bytesutil.SafeCopyBytes(header.StateRoot)
	bodyRoot := bytesutil.SafeCopyBytes(header.BodyRoot)
	return &BeaconBlockHeader{
		Slot:          header.Slot,
		ProposerIndex: header.ProposerIndex,
		ParentRoot:    parentRoot,
		StateRoot:     stateRoot,
		BodyRoot:      bodyRoot,
	}
}

// Copy --
func (deposit *Deposit) Copy() *Deposit {
	if deposit == nil {
		return nil
	}
	return &Deposit{
		Proof: bytesutil.SafeCopy2dBytes(deposit.Proof),
		Data:  deposit.Data.Copy(),
	}
}

// Copy --
func (depData *Deposit_Data) Copy() *Deposit_Data {
	if depData == nil {
		return nil
	}
	return &Deposit_Data{
		PublicKey:             bytesutil.SafeCopyBytes(depData.PublicKey),
		WithdrawalCredentials: bytesutil.SafeCopyBytes(depData.WithdrawalCredentials),
		Amount:                depData.Amount,
		Signature:             bytesutil.SafeCopyBytes(depData.Signature),
	}
}

// Copy --
func (exit *SignedVoluntaryExit) Copy() *SignedVoluntaryExit {
	if exit == nil {
		return nil
	}
	return &SignedVoluntaryExit{
		Exit:      exit.Exit.Copy(),
		Signature: bytesutil.SafeCopyBytes(exit.Signature),
	}
}

// Copy --
func (exit *VoluntaryExit) Copy() *VoluntaryExit {
	if exit == nil {
		return nil
	}
	return &VoluntaryExit{
		Epoch:          exit.Epoch,
		ValidatorIndex: exit.ValidatorIndex,
	}
}

// Copy --
func (a *SyncAggregate) Copy() *SyncAggregate {
	if a == nil {
		return nil
	}
	return &SyncAggregate{
		SyncCommitteeBits:      bytesutil.SafeCopyBytes(a.SyncCommitteeBits),
		SyncCommitteeSignature: bytesutil.SafeCopyBytes(a.SyncCommitteeSignature),
	}
}

// Copy --
func (change *SignedBLSToExecutionChange) Copy() *SignedBLSToExecutionChange {
	if change == nil {
		return nil
	}
	return &SignedBLSToExecutionChange{
		Message:   change.Message.Copy(),
		Signature: bytesutil.SafeCopyBytes(change.Signature),
	}
}

// Copy --
func (change *BLSToExecutionChange) Copy() *BLSToExecutionChange {
	if change == nil {
		return nil
	}
	return &BLSToExecutionChange{
		ValidatorIndex:     change.ValidatorIndex,
		FromBlsPubkey:      bytesutil.SafeCopyBytes(change.FromBlsPubkey),
		ToExecutionAddress: bytesutil.SafeCopyBytes(change.ToExecutionAddress),
	}
}

// Copy --
func (summary *HistoricalSummary) Copy() *HistoricalSummary {
	if summary == nil {
		return nil
	}
	return &HistoricalSummary{
		BlockSummaryRoot: bytesutil.SafeCopyBytes(summary.BlockSummaryRoot),
		StateSummaryRoot: bytesutil.SafeCopyBytes(summary.StateSummaryRoot),
	}
}

// Copy --
func (pbd *PendingBalanceDeposit) Copy() *PendingBalanceDeposit {
	if pbd == nil {
		return nil
	}
	return &PendingBalanceDeposit{
		Index:  pbd.Index,
		Amount: pbd.Amount,
	}
}
