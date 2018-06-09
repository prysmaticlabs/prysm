// Package params defines important configuration options to be used when instantiating
// objects within the sharding package. For example, it defines objects such as a
// ShardConfig that will be useful when creating new shard instances.
package params

import "github.com/ethereum/go-ethereum/common"

type ShardConfig struct {
	SMCAddress common.Address
}
