package validator

import (
	"bytes"

	eth "github.com/prysmaticlabs/prysm/proto/prysm/v2"
)

type proposerSyncContributions []*eth.SyncCommitteeContribution

// filter separates sync aggregate list into two groups: valid and invalid sync aggregate.
// The valid group contains the input block root.
func (cs proposerSyncContributions) filter(r [32]byte) proposerSyncContributions {
	validSyncContributions := make([]*eth.SyncCommitteeContribution, 0, len(cs))
	for _, c := range cs {
		if bytes.Equal(c.BlockRoot, r[:]) {
			validSyncContributions = append(validSyncContributions, c)
		}
	}
	return validSyncContributions
}

// dedup removes duplicate sync contributions (ones with the same bits set on).
// Important: not only exact duplicates are removed, but proper subsets are removed too
// (their known bits are redundant and are already contained in their supersets).
func (cs proposerSyncContributions) dedup() proposerSyncContributions {
	if len(cs) < 2 {
		return cs
	}
	contributionsBySubIdx := make(map[uint64][]*eth.SyncCommitteeContribution, len(cs))
	for _, c := range cs {
		contributionsBySubIdx[c.SubcommitteeIndex] = append(contributionsBySubIdx[c.SubcommitteeIndex], c)
	}

	uniqContributions := make([]*eth.SyncCommitteeContribution, 0, len(cs))
	for _, cs := range contributionsBySubIdx {
		for i := 0; i < len(cs); i++ {
			_ = cs[i]
			for j := i + 1; i < len(cs); j++ {
				_ = cs[j]
				//TODO: Implement bit vector contains.
			}
		}
	}

	return uniqContributions
}
