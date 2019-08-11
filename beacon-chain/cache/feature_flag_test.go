package cache

import "github.com/prysmaticlabs/prysm/shared/featureconfig"

func init() {
	featureconfig.InitFeatureConfig(&featureconfig.FeatureFlagConfig{
		EnableAttestationCache:   true,
		EnableActiveBalanceCache: true,
		EnableAncestorBlockCache: true,
		EnableStartShardCache:    true,
		EnableSeedCache:          true,
		EnableEth1DataVoteCache:  true,
		EnableTotalBalanceCache:  true,
	})
}
