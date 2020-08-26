package params

import (
	"time"

	"github.com/mohae/deepcopy"
)

func init() {
	// Using medalla as the default configuration for now.
	UseMedallaNetworkConfig()
}

// NetworkConfig defines the spec based network parameters.
type NetworkConfig struct {
	GossipMaxSize                     uint64        `yaml:"GOSSIP_MAX_SIZE"`                       // GossipMaxSize is the maximum allowed size of uncompressed gossip messages.
	MaxChunkSize                      uint64        `yaml:"MAX_CHUNK_SIZE"`                        // MaxChunkSize is the the maximum allowed size of uncompressed req/resp chunked responses.
	AttestationSubnetCount            uint64        `yaml:"ATTESTATION_SUBNET_COUNT"`              // AttestationSubnetCount is the number of attestation subnets used in the gossipsub protocol.
	AttestationPropagationSlotRange   uint64        `yaml:"ATTESTATION_PROPAGATION_SLOT_RANGE"`    // AttestationPropagationSlotRange is the maximum number of slots during which an attestation can be propagated.
	RandomSubnetsPerValidator         uint64        `yaml:"RANDOM_SUBNETS_PER_VALIDATOR"`          // RandomSubnetsPerValidator specifies the amount of subnets a validator has to be subscribed to at one time.
	EpochsPerRandomSubnetSubscription uint64        `yaml:"EPOCHS_PER_RANDOM_SUBNET_SUBSCRIPTION"` // EpochsPerRandomSubnetSubscription specifies the minimum duration a validator is connected to their subnet.
	MaxRequestBlocks                  uint64        `yaml:"MAX_REQUEST_BLOCKS"`                    // MaxRequestBlocks is the maximum number of blocks in a single request.
	TtfbTimeout                       time.Duration `yaml:"TTFB_TIMEOUT"`                          // TtfbTimeout is the maximum time to wait for first byte of request response (time-to-first-byte).
	RespTimeout                       time.Duration `yaml:"RESP_TIMEOUT"`                          // RespTimeout is the maximum time for complete response transfer.
	MaximumGossipClockDisparity       time.Duration `yaml:"MAXIMUM_GOSSIP_CLOCK_DISPARITY"`        // MaximumGossipClockDisparity is the maximum milliseconds of clock disparity assumed between honest nodes.

	// DiscoveryV5 Config
	ETH2Key      string // ETH2Key is the ENR key of the eth2 object in an enr.
	AttSubnetKey string // AttSubnetKey is the ENR key of the subnet bitfield in the enr.

	// Chain Network Config
	ContractDeploymentBlock uint64   // ContractDeploymentBlock is the eth1 block in which the deposit contract is deployed.
	DepositContractAddress  string   // DepositContractAddress is the address of the deposit contract.
	ChainID                 uint64   // ChainID of the eth1 network. This used for replay protection.
	NetworkID               uint64   // NetworkID of the eth1 network. This used for replay protection.
	BootstrapNodes          []string // BootstrapNodes are the addresses of the bootnodes.
}

var networkConfig = mainnetNetworkConfig

// BeaconNetworkConfig returns the current network config for
// the beacon chain.
func BeaconNetworkConfig() *NetworkConfig {
	return networkConfig
}

// OverrideBeaconNetworkConfig will override the network
// config with the added argument.
func OverrideBeaconNetworkConfig(cfg *NetworkConfig) {
	networkConfig = cfg.Copy()
}

// Copy returns Copy of the config object.
func (c *NetworkConfig) Copy() *NetworkConfig {
	config, ok := deepcopy.Copy(*c).(NetworkConfig)
	if !ok {
		config = *networkConfig
	}
	return &config
}
