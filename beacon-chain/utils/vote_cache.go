package utils

// VoteCache is a helper cache to track which validators voted
// for a certain block hash and total deposit supported for such block hash.
type VoteCache struct {
	VoterIndices     []uint32
	VoteTotalDeposit uint64
}

func NewVoteCache() *VoteCache {
	return &VoteCache{VoterIndices: []uint32{}, VoteTotalDeposit: 0}
}

func (v *VoteCache) Copy() *VoteCache {
	voterIndices := make([]uint32, len(v.VoterIndices))
	copy(voterIndices, v.VoterIndices)

	return &VoteCache{
		VoterIndices:     voterIndices,
		VoteTotalDeposit: v.VoteTotalDeposit,
	}
}

func VoteCacheDeepCopy(old map[[32]byte]*VoteCache) map[[32]byte]*VoteCache {
	new := map[[32]byte]*VoteCache{}
	for k, v := range old {
		newK := [32]byte{}
		copy(newK[:], k[:])

		new[newK] = v.Copy()
	}

	return new
}
