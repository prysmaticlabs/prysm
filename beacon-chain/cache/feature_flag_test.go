package cache

import "github.com/prysmaticlabs/prysm/shared/featureconfig"

func init() {
	featureconfig.Init(&featureconfig.Flags{
		EnableEth1DataVoteCache: true,
	})
}
