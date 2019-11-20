package cache

import "github.com/prysmaticlabs/prysm/shared/featureconfig"

func init() {
	featureconfig.Init(&featureconfig.Flags{
		EnableAttestationCache:   true,
		EnableEth1DataVoteCache:  true,
		EnableShuffledIndexCache: true,
		EnableCommitteeCache:     true,
		EnableActiveCountCache:   true,
		EnableActiveIndicesCache: true,
	})
}
