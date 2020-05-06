package params

import "time"

// NetworkConfig defines the spec based network parameters.
type NetworkConfig struct {
	GossipMaxSize                     uint64        `yaml:"GOSSIP_MAX_SIZE"`                       // GossipMaxSize is the maximum allowed size of uncompressed gossip messages.
	MaxChunkSize                      uint64        `yaml:"MAX_CHUNK_SIZE"`                        // MaxChunkSize is the the maximum allowed size of uncompressed req/resp chunked responses.
	AttestationSubnetCount            uint64        `yaml:"ATTESTATION_SUBNET_COUNT"`              // AttestationSubnetCount is the number of attestation subnets used in the gossipsub protocol.
	AttestationPropagationSlotRange   uint64        `yaml:"ATTESTATION_PROPAGATION_SLOT_RANGE"`    // AttestationPropagationSlotRange is the maximum number of slots during which an attestation can be propagated.
	RandomSubnetsPerValidator         uint64        `yaml:"RANDOM_SUBNETS_PER_VALIDATOR"`          // RandomSubnetsPerValidator specifies the amount of subnets a validator has to be subscribed to at one time.
	EpochsPerRandomSubnetSubscription uint64        `yaml:"EPOCHS_PER_RANDOM_SUBNET_SUBSCRIPTION"` // EpochsPerRandomSubnetSubscription specifies the minimum duration a validator is connected to their subnet.
	TtfbTimeout                       time.Duration `yaml:"TTFB_TIMEOUT"`                          // TtfbTimeout is the maximum time to wait for first byte of request response (time-to-first-byte).
	RespTimeout                       time.Duration `yaml:"RESP_TIMEOUT"`                          // RespTimeout is the maximum time for complete response transfer.
	MaximumGossipClockDisparity       time.Duration `yaml:"MAXIMUM_GOSSIP_CLOCK_DISPARITY"`        // MaximumGossipClockDisparity is the maximum milliseconds of clock disparity assumed between honest nodes.

	// DiscoveryV5 Config
	ETH2Key      string // ETH2Key is the ENR key of the eth2 object in an enr.
	AttSubnetKey string // AttSubnetKey is the ENR key of the subnet bitfield in the enr.
}

var defaultNetworkConfig = &NetworkConfig{
	GossipMaxSize:                     1 << 20, // 1 MiB
	MaxChunkSize:                      1 << 20, // 1 MiB
	AttestationSubnetCount:            64,
	AttestationPropagationSlotRange:   32,
	RandomSubnetsPerValidator:         1 << 0,
	EpochsPerRandomSubnetSubscription: 1 << 8,
	TtfbTimeout:                       5 * time.Second,
	RespTimeout:                       10 * time.Second,
	MaximumGossipClockDisparity:       500 * time.Millisecond,
	ETH2Key:                           "eth2",
	AttSubnetKey:                      "attnets",
}

// BeaconNetworkConfig returns the current network config for
// the beacon chain.
func BeaconNetworkConfig() *NetworkConfig {
	return defaultNetworkConfig
}
