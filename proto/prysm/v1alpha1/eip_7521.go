package eth

// Copy --
func (pw *PendingPartialWithdrawal) Copy() *PendingPartialWithdrawal {
	if pw == nil {
		return nil
	}
	return &PendingPartialWithdrawal{
		Index:             pw.Index,
		Amount:            pw.Amount,
		WithdrawableEpoch: pw.WithdrawableEpoch,
	}
}

// Copy --
func (pc *PendingConsolidation) Copy() *PendingConsolidation {
	if pc == nil {
		return nil
	}
	return &PendingConsolidation{
		SourceIndex: pc.SourceIndex,
		TargetIndex: pc.TargetIndex,
	}
}
