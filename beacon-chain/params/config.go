package params

import (
	"github.com/ethereum/go-ethereum/common"
)

const (
	// AttesterCount is the number of attesters per committee/
	AttesterCount = 32
	// AttesterReward determines how much ETH attesters get for performing their duty.
	AttesterReward = 1
	// EpochLength is the beacon chain epoch length in blocks.
	EpochLength = 5
	// ShardCount is a fixed number.
	ShardCount = 20
	// DefaultBalance of a validator.
	DefaultBalance = 32000
	// DefaultSwitchDynasty value.
	DefaultSwitchDynasty = 9999999999999999999
	// MaxValidators in the protocol.
	MaxValidators = 2 ^ 24
	// NotariesPerCrosslink fixed to 100.
	NotariesPerCrosslink = 100
)

// Web3ServiceConfig defines a config struct for a web3 service to use through its life cycle.
type Web3ServiceConfig struct {
	Endpoint string
	Pubkey   string
	VrcAddr  common.Address
}
