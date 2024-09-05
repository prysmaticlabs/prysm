package eth

func (u *LightClientUpdate) HasSupermajority() bool {
	return u.SyncAggregate.SyncCommitteeBits.Count()*3 >= u.SyncAggregate.SyncCommitteeBits.Len()*2
}

func (u *LightClientFinalityUpdate) HasSupermajority() bool {
	return u.SyncAggregate.SyncCommitteeBits.Count()*3 >= u.SyncAggregate.SyncCommitteeBits.Len()*2
}
