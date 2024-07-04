//go:build minimal

package field_params

const (
	Preset                                = "minimal"
	BlockRootsLength                      = 64            // SLOTS_PER_HISTORICAL_ROOT
	StateRootsLength                      = 64            // SLOTS_PER_HISTORICAL_ROOT
	RandaoMixesLength                     = 64            // EPOCHS_PER_HISTORICAL_VECTOR
	HistoricalRootsLength                 = 16777216      // HISTORICAL_ROOTS_LIMIT
	ValidatorRegistryLimit                = 1099511627776 // VALIDATOR_REGISTRY_LIMIT
	Eth1DataVotesLength                   = 32            // SLOTS_PER_ETH1_VOTING_PERIOD
	PreviousEpochAttestationsLength       = 1024          // MAX_ATTESTATIONS * SLOTS_PER_EPOCH
	CurrentEpochAttestationsLength        = 1024          // MAX_ATTESTATIONS * SLOTS_PER_EPOCH
	SlashingsLength                       = 64            // EPOCHS_PER_SLASHINGS_VECTOR
	SyncCommitteeLength                   = 32            // SYNC_COMMITTEE_SIZE
	PTCSize                               = 32            // PTC_SIZE [New in ePBS]
	RootLength                            = 32            // RootLength defines the byte length of a Merkle root.
	BLSSignatureLength                    = 96            // BLSSignatureLength defines the byte length of a BLSSignature.
	BLSPubkeyLength                       = 48            // BLSPubkeyLength defines the byte length of a BLSSignature.
	MaxTxsPerPayloadLength                = 1048576       // MaxTxsPerPayloadLength defines the maximum number of transactions that can be included in a payload.
	MaxBytesPerTxLength                   = 1073741824    // MaxBytesPerTxLength defines the maximum number of bytes that can be included in a transaction.
	FeeRecipientLength                    = 20            // FeeRecipientLength defines the byte length of a fee recipient.
	LogsBloomLength                       = 256           // LogsBloomLength defines the byte length of a logs bloom.
	VersionLength                         = 4             // VersionLength defines the byte length of a fork version number.
	SlotsPerEpoch                         = 8             // SlotsPerEpoch defines the number of slots per epoch.
	SyncCommitteeAggregationBytesLength   = 1             // SyncCommitteeAggregationBytesLength defines the sync committee aggregate bytes.
	SyncAggregateSyncCommitteeBytesLength = 4             // SyncAggregateSyncCommitteeBytesLength defines the length of sync committee bytes in a sync aggregate.
	MaxWithdrawalsPerPayload              = 4             // MaxWithdrawalsPerPayloadLength defines the maximum number of withdrawals that can be included in a payload.
	MaxBlobsPerBlock                      = 6             // MaxBlobsPerBlock defines the maximum number of blobs with respect to consensus rule can be included in a block.
	MaxBlobCommitmentsPerBlock            = 16            // MaxBlobCommitmentsPerBlock defines the theoretical limit of blobs can be included in a block.
	MaxPayloadAttestationsPerBlock        = 4             // MAX_PAYLOAD_ATTESTATIONS [New in ePBS]
	MaxTransactionsPerInclusionList       = 16            // MAX_TRANSACTIONS_PER_INCLUSION_LIST	[New in ePBS]
	LogMaxBlobCommitments                 = 4             // Log_2 of MaxBlobCommitmentsPerBlock
	BlobLength                            = 131072        // BlobLength defines the byte length of a blob.
	BlobSize                              = 131072        // defined to match blob.size in bazel ssz codegen
	KzgCommitmentInclusionProofDepth      = 17            // Merkle proof depth for blob_kzg_commitments list item
	NextSyncCommitteeBranchDepth          = 5             // NextSyncCommitteeBranchDepth defines the depth of the next sync committee branch.
	PendingBalanceDepositsLimit           = 134217728     // Maximum number of pending balance deposits in the beacon state.
	PendingPartialWithdrawalsLimit        = 64            // Maximum number of pending partial withdrawals in the beacon state.
	PendingConsolidationsLimit            = 64            // Maximum number of pending consolidations in the beacon state.
	MaxDepositRequestsPerPayload          = 4             // Maximum number of deposit requests in an execution payload.
	MaxWithdrawalRequestsPerPayload       = 2             // Maximum number of execution layer withdrawal requests in an execution payload.
	MaxConsolidationRequestsPerPayload    = 1             // Maximum number of consolidation requests in an execution payload.
)
