package utils

import (
	"bytes"
	"encoding/gob"
)

// BlockVote is for tracking which validators voted for a certain block hash
// and total deposit supported for such block hash.
type BlockVote struct {
	VoterIndices     []uint32
	VoteTotalDeposit uint64
}

type BlockVoteCache map[[32]byte]*BlockVote

// NewBlockVote generates a fresh new BlockVote.
func NewBlockVote() *BlockVote {
	return &BlockVote{VoterIndices: []uint32{}, VoteTotalDeposit: 0}
}

func (v *BlockVote) Marshal() ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (v *BlockVote) Unmarshal(blob []byte) error {
	buf := bytes.NewBuffer(blob)
	decoder := gob.NewDecoder(buf)
	if err := decoder.Decode(v); err != nil {
		return err
	}
	return nil
}

// Copy copies a vote cache from itself to a new one.
func (v *BlockVote) Copy() *BlockVote {
	voterIndices := make([]uint32, len(v.VoterIndices))
	copy(voterIndices, v.VoterIndices)

	return &BlockVote{
		VoterIndices:     voterIndices,
		VoteTotalDeposit: v.VoteTotalDeposit,
	}
}

func NewBlockVoteCache() BlockVoteCache {
	return make(BlockVoteCache)
}

func (blockVoteCache BlockVoteCache) IsVoteCacheExist(blockHash [32]byte) bool {
	_, ok := blockVoteCache[blockHash]
	return ok
}

// BlockVoteCacheDeepCopy copies the every BlockVote inside the old cache
// and create a new cache
func BlockVoteCacheDeepCopy(old BlockVoteCache) BlockVoteCache {
	new := BlockVoteCache{}
	for k, v := range old {
		newK := [32]byte{}
		copy(newK[:], k[:])

		new[newK] = v.Copy()
	}

	return new
}
