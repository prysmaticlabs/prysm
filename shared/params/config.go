// Package params defines important constants that are essential to the
// Ethereum 2.0 services.
package params

import (
	"math/big"
	"time"
)

// BeaconChainConfig contains constant configs for node to participate in beacon chain.
type BeaconChainConfig struct {
	// Misc constants.
	ShardCount                   uint64 // ShardCount is the number of shard chains in Ethereum 2.0.
	TargetCommitteeSize          uint64 // TargetCommitteeSize is the number of validators in a committee when the chain is healthy.
	MaxBalanceChurnQuotient      uint64 // MaxBalanceChurnQuotient is used to determine how many validators can rotate per epoch.
	BeaconChainShardNumber       uint64 // BeaconChainShardNumber is the shard number of the beacon chain.
	MaxIndicesPerSlashableVote   uint64 // MaxIndicesPerSlashableVote is used to determine how many validators can be slashed per vote.
	LatestBlockRootsLength       uint64 // LatestBlockRootsLength is the number of block roots kept in the beacon state.
	LatestRandaoMixesLength      uint64 // LatestRandaoMixesLength is the number of randao mixes kept in the beacon state.
	LatestSlashedExitLength      uint64 // LatestSlashedExitLength is used to track penalized exit balances per time interval.
	LatestActiveIndexRootsLength uint64 // LatestIndexRootsLength is the number of index roots kept in beacon state, used by light client.
	MaxExitDequeuesPerEpoch      uint64 // MaxWithdrawalsPerEpoch is the max withdrawals can happen for a single epoch.
	ValidatorPrivkeyFileName     string // ValidatorPrivKeyFileName specifies the string name of a validator private key file.
	WithdrawalPrivkeyFileName    string // WithdrawalPrivKeyFileName specifies the string name of a withdrawal private key file.
	BLSPubkeyLength              int    // BLSPubkeyLength defines the expected length of BLS public keys in bytes.
	DefaultBufferSize            int    // DefaultBufferSize for channels across the Prysm repository.
	HashCacheSize                int64  // HashCacheSize defines the size of object hashes that are cached.

	// BLS domain values.
	DomainDeposit     uint64 // DomainDeposit defines the BLS signature domain for deposit verification.
	DomainAttestation uint64 // DomainAttestation defines the BLS signature domain for attestation verification.
	DomainProposal    uint64 // DomainProposal defines the BLS signature domain for proposal verification.
	DomainExit        uint64 // DomainExit defines the BLS signature domain for exit verification.
	DomainRandao      uint64 // DomainRandao defines the BLS signature domain for randao verification.
	DomainTransfer    uint64 // DomainTransfer defines the BLS signature domain for transfer verification.

	// Deposit contract constants.
	DepositContractAddress   []byte // DepositContractAddress is the address of the deposit contract in PoW chain.
	DepositContractTreeDepth uint64 // Depth of the Merkle trie of deposits in the validator deposit contract on the PoW chain.

	// Gwei Values
	MinDepositAmount           uint64 // MinDepositAmount is the maximal amount of Gwei a validator can send to the deposit contract at once.
	MaxDepositAmount           uint64 // MaxDepositAmount is the maximal amount of Gwei a validator can send to the deposit contract at once.
	EjectionBalance            uint64 // EjectionBalance is the minimal GWei a validator needs to have before ejected.
	ForkChoiceBalanceIncrement uint64 // ForkChoiceBalanceIncrement is used to track block score based on balances for fork choice.

	// Initial value constants.
	GenesisForkVersion      uint64   // GenesisForkVersion is used to track fork version between state transitions.
	GenesisSlot             uint64   // GenesisSlot is used to initialize the genesis state fields.
	GenesisEpoch            uint64   // GenesisEpoch is used to initialize epoch.
	GenesisStartShard       uint64   // GenesisStartShard is the first shard to assign validators.
	ZeroHash                [32]byte // ZeroHash is used to represent a zeroed out 32 byte array.
	EmptySignature          [96]byte // EmptySignature is used to represent a zeroed out BLS Signature.
	BLSWithdrawalPrefixByte byte     // BLSWithdrawalPrefixByte is used for BLS withdrawal and it's the first byte.

	// Time parameters constants.
	SecondsPerSlot               uint64 // SecondsPerSlot is how many seconds are in a single slot.
	MinAttestationInclusionDelay uint64 // MinAttestationInclusionDelay defines how long validator has to wait to include attestation for beacon block.
	SlotsPerEpoch                uint64 // SlotsPerEpoch is the number of slots in an epoch.
	MinSeedLookahead             uint64 // SeedLookahead is the duration of randao look ahead seed.
	ActivationExitDelay          uint64 // EntryExitDelay is the duration a validator has to wait for entry and exit in epoch.
	EpochsPerEth1VotingPeriod    uint64 //  defines how often the merkle root of deposit receipts get updated in beacon node.
	Eth1FollowDistance           uint64 // Eth1FollowDistance is the number of eth1.0 blocks to wait before considering a new deposit for voting. This only applies after the chain as been started.
	MinValidatorWithdrawalDelay  uint64 // MinValidatorWithdrawalEpochs is the shortest amount of time a validator can get the deposit out.
	FarFutureEpoch               uint64 // FarFutureEpoch represents a epoch extremely far away in the future used as the default penalization slot for validators.

	// Reward and penalty quotients constants.
	BaseRewardQuotient                 uint64 // BaseRewardQuotient is used to calculate validator per-slot interest rate.
	WhistlerBlowerRewardQuotient       uint64 // WhistlerBlowerRewardQuotient is used to calculate whistler blower reward.
	AttestationInclusionRewardQuotient uint64 // IncluderRewardQuotient defines the reward quotient of proposer for including attestations..
	InactivityPenaltyQuotient          uint64 // InactivityPenaltyQuotient defines how much validator leaks out balances for offline.
	GweiPerEth                         uint64 // GweiPerEth is the amount of gwei corresponding to 1 eth.

	// Max operations per block constants.
	MaxVoluntaryExits    uint64 // MaxVoluntaryExits determines the maximum number of validator exits in a block.
	MaxDeposits          uint64 // MaxVoluntaryExits determines the maximum number of validator deposits in a block.
	MaxAttestations      uint64 // MaxAttestations defines the maximum allowed attestations in a beacon block.
	MaxProposerSlashings uint64 // MaxProposerSlashings defines the maximum number of slashings of proposers possible in a block.
	MaxAttesterSlashings uint64 // MaxAttesterSlashings defines the maximum number of casper FFG slashings possible in a block.

	// Prysm constants.
	DepositsForChainStart   uint64        // DepositsForChainStart defines how many validator deposits needed to kick off beacon chain.
	RandBytes               uint64        // RandBytes is the number of bytes used as entropy to shuffle validators.
	SyncPollingInterval     int64         // SyncPollingInterval queries network nodes for sync status.
	BatchBlockLimit         uint64        // BatchBlockLimit is maximum number of blocks that can be requested for initial sync.
	SyncEpochLimit          uint64        // SyncEpochLimit is the number of epochs the current node can be behind before it requests for the latest state.
	MaxNumLog2Validators    uint64        // MaxNumLog2Validators is the Max number of validators in Log2 exists given total ETH supply.
	LogBlockDelay           int64         // Number of blocks to wait from the current head before processing logs from the deposit contract.
	RPCSyncCheck            time.Duration // Number of seconds to query the sync service, to find out if the node is synced or not.
	TestnetContractEndpoint string        // TestnetContractEndpoint to fetch the contract address of the Prysmatic Labs testnet.
	GoerliBlockTime         uint64        // GoerliBlockTime is the number of seconds on avg a Goerli block is created.
}

// DepositContractConfig contains the deposits for
type DepositContractConfig struct {
	DepositsForChainStart *big.Int // DepositsForChainStart defines how many validator deposits needed to kick off beacon chain.
	MinDepositAmount      *big.Int // MinDepositAmount defines the minimum deposit amount in gwei that is required in the deposit contract.
	MaxDepositAmount      *big.Int // MaxDepositAmount defines the maximum deposit amount in gwei that is required in the deposit contract.
}

// ShardChainConfig contains configs for node to participate in shard chains.
type ShardChainConfig struct {
	ChunkSize         uint64 // ChunkSize defines the size of each chunk in bytes.
	MaxShardBlockSize uint64 // MaxShardBlockSize defines the max size of each shard block in bytes.
}

var defaultBeaconConfig = &BeaconChainConfig{
	// Misc constant.
	ShardCount:                   1024,
	TargetCommitteeSize:          128,
	MaxBalanceChurnQuotient:      32,
	BeaconChainShardNumber:       1<<64 - 1,
	MaxIndicesPerSlashableVote:   4096,
	LatestBlockRootsLength:       8192,
	LatestRandaoMixesLength:      8192,
	LatestSlashedExitLength:      8192,
	LatestActiveIndexRootsLength: 8192,
	MaxExitDequeuesPerEpoch:      4,
	ValidatorPrivkeyFileName:     "/validatorprivatekey",
	WithdrawalPrivkeyFileName:    "/shardwithdrawalkey",
	BLSPubkeyLength:              96,
	DefaultBufferSize:            10000,
	HashCacheSize:                100000,

	// BLS domain values.
	DomainDeposit:     0,
	DomainAttestation: 1,
	DomainProposal:    2,
	DomainExit:        3,
	DomainRandao:      4,
	DomainTransfer:    5,

	// Deposit contract constants.
	DepositContractTreeDepth: 32,

	// Gwei values:
	MinDepositAmount:           1 * 1e9,
	MaxDepositAmount:           32 * 1e9,
	EjectionBalance:            16 * 1e9,
	ForkChoiceBalanceIncrement: 1 * 1e9,

	// Initial value constants.
	GenesisForkVersion:      0,
	GenesisSlot:             1 << 63,
	GenesisEpoch:            1 << 63 / 64,
	GenesisStartShard:       0,
	FarFutureEpoch:          1<<64 - 1,
	ZeroHash:                [32]byte{},
	EmptySignature:          [96]byte{},
	BLSWithdrawalPrefixByte: byte(0),

	// Time parameter constants.
	SecondsPerSlot:               6,
	MinAttestationInclusionDelay: 4,
	SlotsPerEpoch:                64,
	MinSeedLookahead:             1,
	ActivationExitDelay:          4,
	EpochsPerEth1VotingPeriod:    16,
	Eth1FollowDistance:           1024,

	// Reward and penalty quotients constants.
	BaseRewardQuotient:                 32,
	WhistlerBlowerRewardQuotient:       512,
	AttestationInclusionRewardQuotient: 8,
	InactivityPenaltyQuotient:          1 << 24,
	GweiPerEth:                         1000000000,

	// Max operations per block constants.
	MaxVoluntaryExits:    16,
	MaxDeposits:          16,
	MaxAttestations:      128,
	MaxProposerSlashings: 16,
	MaxAttesterSlashings: 1,

	// Prysm constants.
	DepositsForChainStart: 16384,
	RandBytes:             3,
	BatchBlockLimit:       64 * 4, // Process blocks in batches of 4 epochs of blocks (threshold before casper penalties).
	MaxNumLog2Validators:  24,
	LogBlockDelay:         2, //
	RPCSyncCheck:          1,
	GoerliBlockTime:       14, // 14 seconds on average for a goerli block to be created.

	// Testnet misc values.
	TestnetContractEndpoint: "https://beta.prylabs.net/contract", // defines an http endpoint to fetch the testnet contract addr.
}

var defaultShardConfig = &ShardChainConfig{
	ChunkSize:         uint64(256),
	MaxShardBlockSize: uint64(32768),
}

var defaultDepositContractConfig = &DepositContractConfig{
	DepositsForChainStart: big.NewInt(16384),
	MinDepositAmount:      big.NewInt(1e9),
	MaxDepositAmount:      big.NewInt(32e9),
}

var beaconConfig = defaultBeaconConfig
var shardConfig = defaultShardConfig
var contractConfig = defaultDepositContractConfig

// BeaconConfig retrieves beacon chain config.
func BeaconConfig() *BeaconChainConfig {
	return beaconConfig
}

// DemoBeaconConfig retrieves the demo beacon chain config.
func DemoBeaconConfig() *BeaconChainConfig {
	demoConfig := *defaultBeaconConfig
	demoConfig.ShardCount = 1
	demoConfig.MinAttestationInclusionDelay = 1
	demoConfig.TargetCommitteeSize = 1
	demoConfig.DepositsForChainStart = 8
	demoConfig.SlotsPerEpoch = 8
	demoConfig.GenesisEpoch = demoConfig.GenesisSlot / 8
	demoConfig.MinDepositAmount = 100
	demoConfig.MaxDepositAmount = 3.2 * 1e9
	demoConfig.EjectionBalance = 3.175 * 1e9
	demoConfig.SyncPollingInterval = 1 * 10 // Query nodes over the network every slot.
	demoConfig.Eth1FollowDistance = 5
	demoConfig.EpochsPerEth1VotingPeriod = 1
	demoConfig.LatestRandaoMixesLength = 5 * demoConfig.SlotsPerEpoch
	demoConfig.LatestActiveIndexRootsLength = 5 * demoConfig.SlotsPerEpoch
	demoConfig.LatestSlashedExitLength = 5 * demoConfig.SlotsPerEpoch
	demoConfig.LatestBlockRootsLength = 5 * demoConfig.SlotsPerEpoch

	return &demoConfig
}

// ShardConfig retrieves shard chain config.
func ShardConfig() *ShardChainConfig {
	return shardConfig
}

// ContractConfig retrieves the deposit contract config
func ContractConfig() *DepositContractConfig {
	return contractConfig
}

// DemoContractConfig uses the argument provided to initialize a fresh config.
func DemoContractConfig(depositsReq *big.Int, minDeposit *big.Int, maxDeposit *big.Int) *DepositContractConfig {
	return &DepositContractConfig{
		DepositsForChainStart: depositsReq,
		MinDepositAmount:      minDeposit,
		MaxDepositAmount:      maxDeposit,
	}
}

// UseDemoBeaconConfig for beacon chain services.
func UseDemoBeaconConfig() {
	beaconConfig = DemoBeaconConfig()
}

// OverrideBeaconConfig by replacing the config. The preferred pattern is to
// call BeaconConfig(), change the specific parameters, and then call
// OverrideBeaconConfig(c). Any subsequent calls to params.BeaconConfig() will
// return this new configuration.
func OverrideBeaconConfig(c *BeaconChainConfig) {
	beaconConfig = c
}
