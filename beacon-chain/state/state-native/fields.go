package state_native

import (
	"fmt"
	"slices"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/state/state-native/types"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/encoding/ssz"
	"github.com/OffchainLabs/prysm/v7/runtime/version"
)

var (
	phase0Fields = []types.FieldIndex{
		types.GenesisTime,
		types.GenesisValidatorsRoot,
		types.Slot,
		types.Fork,
		types.LatestBlockHeader,
		types.BlockRoots,
		types.StateRoots,
		types.HistoricalRoots,
		types.Eth1Data,
		types.Eth1DataVotes,
		types.Eth1DepositIndex,
		types.Validators,
		types.Balances,
		types.RandaoMixes,
		types.Slashings,
		types.PreviousEpochAttestations,
		types.CurrentEpochAttestations,
		types.JustificationBits,
		types.PreviousJustifiedCheckpoint,
		types.CurrentJustifiedCheckpoint,
		types.FinalizedCheckpoint,
	}

	altairFields = []types.FieldIndex{
		types.GenesisTime,
		types.GenesisValidatorsRoot,
		types.Slot,
		types.Fork,
		types.LatestBlockHeader,
		types.BlockRoots,
		types.StateRoots,
		types.HistoricalRoots,
		types.Eth1Data,
		types.Eth1DataVotes,
		types.Eth1DepositIndex,
		types.Validators,
		types.Balances,
		types.RandaoMixes,
		types.Slashings,
		types.PreviousEpochParticipationBits,
		types.CurrentEpochParticipationBits,
		types.JustificationBits,
		types.PreviousJustifiedCheckpoint,
		types.CurrentJustifiedCheckpoint,
		types.FinalizedCheckpoint,
		types.InactivityScores,
		types.CurrentSyncCommittee,
		types.NextSyncCommittee,
	}

	bellatrixFields = append(altairFields, types.LatestExecutionPayloadHeader)

	withdrawalAndHistoricalSummaryFields = []types.FieldIndex{
		types.NextWithdrawalIndex,
		types.NextWithdrawalValidatorIndex,
		types.HistoricalSummaries,
	}

	capellaFields = slices.Concat(
		altairFields,
		[]types.FieldIndex{types.LatestExecutionPayloadHeaderCapella},
		withdrawalAndHistoricalSummaryFields,
	)

	denebFields = slices.Concat(
		altairFields,
		[]types.FieldIndex{types.LatestExecutionPayloadHeaderDeneb},
		withdrawalAndHistoricalSummaryFields,
	)

	electraAdditionalFields = []types.FieldIndex{
		types.DepositRequestsStartIndex,
		types.DepositBalanceToConsume,
		types.ExitBalanceToConsume,
		types.EarliestExitEpoch,
		types.ConsolidationBalanceToConsume,
		types.EarliestConsolidationEpoch,
		types.PendingDeposits,
		types.PendingPartialWithdrawals,
		types.PendingConsolidations,
	}

	electraFields = slices.Concat(
		denebFields,
		electraAdditionalFields,
	)

	fuluFields = append(
		electraFields,
		types.ProposerLookahead,
	)

	gloasAdditionalFields = []types.FieldIndex{
		types.Builders,
		types.NextWithdrawalBuilderIndex,
		types.ExecutionPayloadAvailability,
		types.BuilderPendingPayments,
		types.BuilderPendingWithdrawals,
		types.LatestExecutionPayloadBid,
		types.PayloadExpectedWithdrawals,
		types.PTCWindow,
	}

	decoupledAdditionalFields = []types.FieldIndex{
		types.Height,
		types.Entry,
		types.TargetParticipation,
		types.Progress,
		types.JustifiedHeight,
		types.Justified,
		types.FinalizedHeight,
		types.Finalized,
		types.PendingFinality,
	}

	eip8453AdditionalFields = []types.FieldIndex{
		types.TargetCheckpoint,
		types.JustifiedCheckpointEIP8453,
		types.FinalizedCheckpointEIP8453,
		types.TargetSlot,
		types.FinalizedSlot,
		types.TargetParticipationEIP8453,
		types.ProgressParticipation,
		types.RoundParticipation,
		types.PendingFinalities,
		types.HeadVoteParticipation,
	}

	// Removed fields keep their positions as gaps so later fields keep their generalized indices.
	eip8453InactiveFields = []types.FieldIndex{
		types.Eth1Data,
		types.Eth1DataVotes,
		types.Eth1DepositIndex,
		types.PreviousEpochParticipationBits,
		types.CurrentEpochParticipationBits,
		types.JustificationBits,
		types.PreviousJustifiedCheckpoint,
		types.CurrentJustifiedCheckpoint,
		types.FinalizedCheckpoint,
		types.DepositRequestsStartIndex,
	}

	gloasProgressiveSchema     *ProgressiveStateSchema
	gloasFields                []types.FieldIndex
	decoupledProgressiveSchema *ProgressiveStateSchema
	decoupledFields            []types.FieldIndex
	eip8453ProgressiveSchema   *ProgressiveStateSchema
	eip8453Fields              []types.FieldIndex
)

func init() {
	var err error
	gloasProgressiveSchema, err = newProgressiveStateFieldsSchema(
		slices.Concat(
			altairFields,
			[]types.FieldIndex{types.LatestBlockHash},
			withdrawalAndHistoricalSummaryFields,
			electraAdditionalFields,
			[]types.FieldIndex{types.ProposerLookahead},
			gloasAdditionalFields,
		),
		nil,
		params.BeaconConfig().BeaconStateGloasFieldCount,
	)
	if err != nil {
		panic(err)
	}
	gloasFields = gloasProgressiveSchema.fields

	decoupledProgressiveSchema, err = newProgressiveStateFieldsSchema(
		slices.Concat(
			altairFields,
			[]types.FieldIndex{types.LatestBlockHash},
			withdrawalAndHistoricalSummaryFields,
			electraAdditionalFields,
			[]types.FieldIndex{types.ProposerLookahead},
			gloasAdditionalFields,
			decoupledAdditionalFields,
		),
		nil,
		params.BeaconConfig().BeaconStateDecoupledFieldCount,
	)
	if err != nil {
		panic(err)
	}
	decoupledFields = decoupledProgressiveSchema.fields

	eip8453ProgressiveSchema, err = newProgressiveStateFieldsSchema(
		slices.Concat(
			altairFields,
			[]types.FieldIndex{types.LatestBlockHash},
			withdrawalAndHistoricalSummaryFields,
			electraAdditionalFields,
			[]types.FieldIndex{types.ProposerLookahead},
			gloasAdditionalFields,
			eip8453AdditionalFields,
		),
		eip8453InactiveFields,
		params.BeaconConfig().BeaconStateEIP8453FieldCount,
	)
	if err != nil {
		panic(err)
	}
	eip8453Fields = eip8453ProgressiveSchema.fields
}

func ProgressiveStateSchemaForVersion(v int) (*ProgressiveStateSchema, bool) {
	switch v {
	case version.Gloas:
		return gloasProgressiveSchema, true
	case version.Decoupled:
		return decoupledProgressiveSchema, true
	case version.EIP8453:
		return eip8453ProgressiveSchema, true
	default:
		return nil, false
	}
}

type ProgressiveStateSchema struct {
	// activeFields uses stable, append-only field positions and retains false
	// entries for fields removed in later forks.
	activeFields []bool
	fields       []types.FieldIndex
	// Positions count the inactive fields, so each is the field's leaf in the progressive Merkle tree.
	fieldIndex map[types.FieldIndex]int
}

func newProgressiveStateFieldsSchema(allFields []types.FieldIndex, inactiveFields []types.FieldIndex, forkFieldCount int) (*ProgressiveStateSchema, error) {
	if len(allFields) == 0 {
		return nil, fmt.Errorf("progressive state schema requires at least one field")
	}
	if len(allFields) > ssz.MaxProgressiveActiveFields {
		return nil, fmt.Errorf("progressive state schema has %d fields, maximum is %d", len(allFields), ssz.MaxProgressiveActiveFields)
	}

	knownFields := make(map[types.FieldIndex]struct{}, len(allFields))
	for _, field := range allFields {
		if _, exists := knownFields[field]; exists {
			return nil, fmt.Errorf("progressive state schema contains duplicate field %s", field.String())
		}
		knownFields[field] = struct{}{}
	}

	inactive := make(map[types.FieldIndex]struct{}, len(inactiveFields))
	for _, field := range inactiveFields {
		if _, exists := knownFields[field]; !exists {
			return nil, fmt.Errorf("inactive field %s is not present in progressive state schema", field.String())
		}
		if _, exists := inactive[field]; exists {
			return nil, fmt.Errorf("progressive state schema contains duplicate inactive field %s", field.String())
		}
		inactive[field] = struct{}{}
	}

	fields := make([]types.FieldIndex, 0, len(allFields)-len(inactiveFields))
	fieldIndex := make(map[types.FieldIndex]int, len(allFields)-len(inactiveFields))
	activeFields := make([]bool, 0, len(allFields))

	for _, field := range allFields {
		if _, isInactive := inactive[field]; isInactive {
			activeFields = append(activeFields, false)
			continue
		}
		fields = append(fields, field)
		fieldIndex[field] = len(activeFields)
		activeFields = append(activeFields, true)
	}

	if !activeFields[len(activeFields)-1] {
		return nil, fmt.Errorf("progressive state schema active fields must not end with an inactive field")
	}
	if len(fields) != forkFieldCount {
		return nil, fmt.Errorf("fields count does not match expected count: got %d, expected %d", len(fields), forkFieldCount)
	}

	schema := &ProgressiveStateSchema{
		activeFields: activeFields,
		fields:       fields,
		fieldIndex:   fieldIndex,
	}

	return schema, nil
}

func (s *ProgressiveStateSchema) GetFieldIndex(field types.FieldIndex) (int, bool) {
	index, ok := s.fieldIndex[field]
	return index, ok
}

func (s *ProgressiveStateSchema) ActiveFields() []bool {
	return s.activeFields
}

func (s *ProgressiveStateSchema) Fields() []types.FieldIndex {
	return s.fields
}
