package cache

import "github.com/prysmaticlabs/prysm/shared/featureconfig"

func init() {
	featureconfig.InitFeatureConfig(&featureconfig.FeatureFlagConfig{
		EnableAttestationCache:   true,
		EnableActiveBalanceCache: true,
		EnableAncestorBlockCache: true,
		EnableEth1DataVoteCache:  true,
	})
}
