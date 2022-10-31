package params

import (
	"time"

	"github.com/mohae/deepcopy"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
)

// NetworkConfig defines the spec based network parameters.
type NetworkConfig struct {
	ETH2Key                         string        // ETH2Key is the ENR key of the Ethereum consensus object in an enr.
	SyncCommsSubnetKey              string        // SyncCommsSubnetKey is the ENR key of the sync committee subnet bitfield in the enr.
	AttSubnetKey                    string        // AttSubnetKey is the ENR key of the subnet bitfield in the enr.
	BootstrapNodes                  []string      // BootstrapNodes are the addresses of the bootnodes.
	AttestationSubnetCount          uint64        `yaml:"ATTESTATION_SUBNET_COUNT"`       // AttestationSubnetCount is the number of attestation subnets used in the gossipsub protocol.
	MaxChunkSize                    uint64        `yaml:"MAX_CHUNK_SIZE"`                 // MaxChunkSize is the maximum allowed size of uncompressed req/resp chunked responses.
	MaxRequestBlocks                uint64        `yaml:"MAX_REQUEST_BLOCKS"`             // MaxRequestBlocks is the maximum number of blocks in a single request.
	TtfbTimeout                     time.Duration `yaml:"TTFB_TIMEOUT"`                   // TtfbTimeout is the maximum time to wait for first byte of request response (time-to-first-byte).
	RespTimeout                     time.Duration `yaml:"RESP_TIMEOUT"`                   // RespTimeout is the maximum time for complete response transfer.
	MaximumGossipClockDisparity     time.Duration `yaml:"MAXIMUM_GOSSIP_CLOCK_DISPARITY"` // MaximumGossipClockDisparity is the maximum milliseconds of clock disparity assumed between honest nodes.
	GossipMaxSizeBellatrix          uint64        `yaml:"GOSSIP_MAX_SIZE_BELLATRIX"`      // GossipMaxSizeBellatrix is the maximum allowed size of uncompressed gossip messages after the bellatrix epoch.
	ContractDeploymentBlock         uint64        // ContractDeploymentBlock is the eth1 block in which the deposit contract is deployed.
	GossipMaxSize                   uint64        `yaml:"GOSSIP_MAX_SIZE"`                    // GossipMaxSize is the maximum allowed size of uncompressed gossip messages.
	MaxChunkSizeBellatrix           uint64        `yaml:"MAX_CHUNK_SIZE_BELLATRIX"`           // MaxChunkSizeBellatrix is the maximum allowed size of uncompressed req/resp chunked responses after the bellatrix epoch.
	AttestationPropagationSlotRange types.Slot    `yaml:"ATTESTATION_PROPAGATION_SLOT_RANGE"` // AttestationPropagationSlotRange is the maximum number of slots during which an attestation can be propagated.
	MinimumPeersInSubnetSearch      uint64        // PeersInSubnetSearch is the required amount of peers that we need to be able to lookup in a subnet search.
	MessageDomainValidSnappy        [4]byte       `yaml:"MESSAGE_DOMAIN_VALID_SNAPPY"`   // MessageDomainInvalidSnappy is the 4-byte domain for gossip message-id isolation of invalid snappy messages.
	MessageDomainInvalidSnappy      [4]byte       `yaml:"MESSAGE_DOMAIN_INVALID_SNAPPY"` // MessageDomainValidSnappy is the 4-byte domain for gossip message-id isolation of valid snappy messages.
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
