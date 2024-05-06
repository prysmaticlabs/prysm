package eth

func (c *Consolidation) ToPendingConsolidation() *PendingConsolidation {
	if c == nil {
		return nil
	}
	p := &PendingConsolidation{
		SourceIndex: c.SourceIndex,
		TargetIndex: c.TargetIndex,
	}
	return p
}
