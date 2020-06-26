package params

import (
	"time"

	"github.com/mohae/deepcopy"
)

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
	BootstrapNodes          []string // BootstrapNodes are the addresses of the bootnodes.
}

var defaultNetworkConfig = &NetworkConfig{
	GossipMaxSize:                     1 << 20, // 1 MiB
	MaxChunkSize:                      1 << 20, // 1 MiB
	AttestationSubnetCount:            64,
	AttestationPropagationSlotRange:   32,
	RandomSubnetsPerValidator:         1 << 0,
	EpochsPerRandomSubnetSubscription: 1 << 8,
	MaxRequestBlocks:                  1 << 10, // 1024
	TtfbTimeout:                       5 * time.Second,
	RespTimeout:                       10 * time.Second,
	MaximumGossipClockDisparity:       500 * time.Millisecond,
	ETH2Key:                           "eth2",
	AttSubnetKey:                      "attnets",
	ContractDeploymentBlock:           2844925,
	DepositContractAddress:            "0x0F0F0fc0530007361933EaB5DB97d09aCDD6C1c8",
	BootstrapNodes:                    onyxBootnodes,
}

// BeaconNetworkConfig returns the current network config for
// the beacon chain.
func BeaconNetworkConfig() *NetworkConfig {
	return defaultNetworkConfig
}

// UseAltonaNetworkConfig uses the Altona specific
// network config.
func UseAltonaNetworkConfig() {
	cfg := BeaconNetworkConfig()
	cfg.ContractDeploymentBlock = 2917810
	cfg.DepositContractAddress = "0x16e82D77882A663454Ef92806b7DeCa1D394810f"
	cfg.BootstrapNodes = altonaBootnodes
	OverrideBeaconNetworkConfig(cfg)
}

// OverrideBeaconNetworkConfig will override the network
// config with the added argument.
func OverrideBeaconNetworkConfig(cfg *NetworkConfig) {
	defaultNetworkConfig = cfg
}

// Copy returns Copy of the config object.
func (c *NetworkConfig) Copy() *NetworkConfig {
	config, ok := deepcopy.Copy(*c).(NetworkConfig)
	if !ok {
		config = *defaultNetworkConfig
	}
	return &config
}

var (
	// ENRs for Onyx Bootnodes
	onyxBootnodes = []string{"enr:-Ku4QMKVC_MowDsmEa20d5uGjrChI0h8_KsKXDmgVQbIbngZV0idV6_RL7fEtZGo-kTNZ5o7_EJI_vCPJ6scrhwX0Z4Bh2F0dG5ldHOIAAAAAAAAAACEZXRoMpD1pf1CAAAAAP__________gmlkgnY0gmlwhBLf22SJc2VjcDI1NmsxoQJxCnE6v_x2ekgY_uoE1rtwzvGy40mq9eD66XfHPBWgIIN1ZHCCD6A"}
	// ENRs for Altona Bootnodes
	altonaBootnodes = []string{"enr:-LK4QFtV7Pz4reD5a7cpfi1z6yPrZ2I9eMMU5mGQpFXLnLoKZW8TXvVubShzLLpsEj6aayvVO1vFx-MApijD3HLPhlECh2F0dG5ldHOIAAAAAAAAAACEZXRoMpD6etXjAAABIf__________gmlkgnY0gmlwhDMPYfCJc2VjcDI1NmsxoQIerw_qBc9apYfZqo2awiwS930_vvmGnW2psuHsTzrJ8YN0Y3CCIyiDdWRwgiMo",
		"enr:-LK4QPVkFd_MKzdW0219doTZryq40tTe8rwWYO75KDmeZM78fBskGsfCuAww9t8y3u0Q0FlhXOhjE1CWpx3SGbUaU80Ch2F0dG5ldHOIAAAAAAAAAACEZXRoMpD6etXjAAABIf__________gmlkgnY0gmlwhDMPRgeJc2VjcDI1NmsxoQNHu-QfNgzl8VxbMiPgv6wgAljojnqAOrN18tzJMuN8oYN0Y3CCIyiDdWRwgiMo"}
)
