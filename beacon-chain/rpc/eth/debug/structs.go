package debug

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"
	beaconState "github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	enginev1 "github.com/prysmaticlabs/prysm/v4/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

var errPayloadHeaderNotFound = errors.New("expected payload header not found")

type GetBeaconStateV2Response struct {
	Version             string          `json:"version"`
	ExecutionOptimistic bool            `json:"execution_optimistic"`
	Finalized           bool            `json:"finalized"`
	Data                json.RawMessage `json:"data"` // represents the state values based on the version
}

type BeaconState struct {
	GenesisTime                 string                    `json:"genesis_time"`
	GenesisValidatorsRoot       string                    `json:"genesis_validators_root"`
	Slot                        string                    `json:"slot"`
	Fork                        *shared.Fork              `json:"fork"`
	LatestBlockHeader           *shared.BeaconBlockHeader `json:"latest_block_header"`
	BlockRoots                  []string                  `json:"block_roots"`
	StateRoots                  []string                  `json:"state_roots"`
	HistoricalRoots             []string                  `json:"historical_roots"`
	Eth1Data                    *shared.Eth1Data          `json:"eth1_data"`
	Eth1DataVotes               []*shared.Eth1Data        `json:"eth1_data_votes"`
	Eth1DepositIndex            string                    `json:"eth1_deposit_index"`
	Validators                  []*Validator              `json:"validators"`
	Balances                    []string                  `json:"balances"`
	RandaoMixes                 []string                  `json:"randao_mixes"`
	Slashings                   []string                  `json:"slashings"`
	PreviousEpochAttestations   []*PendingAttestation     `json:"previous_epoch_attestations"`
	CurrentEpochAttestations    []*PendingAttestation     `json:"current_epoch_attestations"`
	JustificationBits           string                    `json:"justification_bits"`
	PreviousJustifiedCheckpoint *shared.Checkpoint        `json:"previous_justified_checkpoint"`
	CurrentJustifiedCheckpoint  *shared.Checkpoint        `json:"current_justified_checkpoint"`
	FinalizedCheckpoint         *shared.Checkpoint        `json:"finalized_checkpoint"`
}

func BeaconStateFromConsensus(st beaconState.BeaconState) (*BeaconState, error) {
	if st == nil {
		return nil, errors.New("state is empty")
	}

	f, err := shared.ForkFromConsensus(st.Fork())
	if err != nil {
		return nil, err
	}
	srcbr := st.BlockRoots()
	br := make([]string, len(srcbr))
	for i, r := range srcbr {
		br[i] = hexutil.Encode(r)
	}
	srcsr := st.StateRoots()
	sr := make([]string, len(srcsr))
	for i, r := range srcsr {
		sr[i] = hexutil.Encode(r)
	}
	srchr, err := st.HistoricalRoots()
	if err != nil {
		return nil, err
	}
	hr := make([]string, len(srchr))
	for i, r := range srchr {
		hr[i] = hexutil.Encode(r)
	}
	e1d, err := shared.Eth1DataFromConsensus(st.Eth1Data())
	if err != nil {
		return nil, err
	}
	srcvotes := st.Eth1DataVotes()
	votes := make([]*shared.Eth1Data, len(srcvotes))
	for i, e := range srcvotes {
		votes[i], err = shared.Eth1DataFromConsensus(e)
		if err != nil {
			return nil, err
		}
	}
	srcvals := st.Validators()
	vals := make([]*Validator, len(srcvals))
	for i, v := range srcvals {
		vals[i], err = ValidatorFromConsensus(v)
		if err != nil {
			return nil, err
		}
	}
	srcbals := st.Balances()
	bals := make([]string, len(srcbals))
	for i, b := range srcbals {
		bals[i] = fmt.Sprintf("%d", b)
	}
	srcrm := st.RandaoMixes()
	rm := make([]string, len(srcrm))
	for i, m := range srcrm {
		rm[i] = hexutil.Encode(m)
	}
	srcslashings := st.Slashings()
	slashings := make([]string, len(srcslashings))
	for i, s := range srcslashings {
		slashings[i] = fmt.Sprintf("%d", s)
	}
	srcPrevAtts, err := st.PreviousEpochAttestations()
	if err != nil {
		return nil, err
	}
	prevAtts := make([]*PendingAttestation, len(srcPrevAtts))
	for i, a := range srcPrevAtts {
		prevAtts[i], err = PendingAttestationFromConsensus(a)
		if err != nil {
			return nil, err
		}
	}
	srcCurrAtts, err := st.CurrentEpochAttestations()
	if err != nil {
		return nil, err
	}
	currAtts := make([]*PendingAttestation, len(srcCurrAtts))
	for i, a := range srcCurrAtts {
		currAtts[i], err = PendingAttestationFromConsensus(a)
		if err != nil {
			return nil, err
		}
	}

	return &BeaconState{
		GenesisTime:                 fmt.Sprintf("%d", st.GenesisTime()),
		GenesisValidatorsRoot:       hexutil.Encode(st.GenesisValidatorsRoot()),
		Slot:                        fmt.Sprintf("%d", st.Slot()),
		Fork:                        f,
		LatestBlockHeader:           shared.BeaconBlockHeaderFromConsensus(st.LatestBlockHeader()),
		BlockRoots:                  br,
		StateRoots:                  sr,
		HistoricalRoots:             hr,
		Eth1Data:                    e1d,
		Eth1DataVotes:               votes,
		Eth1DepositIndex:            fmt.Sprintf("%d", st.Eth1DepositIndex()),
		Validators:                  vals,
		Balances:                    bals,
		RandaoMixes:                 rm,
		Slashings:                   slashings,
		PreviousEpochAttestations:   prevAtts,
		CurrentEpochAttestations:    currAtts,
		JustificationBits:           hexutil.Encode(st.JustificationBits()),
		PreviousJustifiedCheckpoint: shared.CheckpointFromConsensus(st.PreviousJustifiedCheckpoint()),
		CurrentJustifiedCheckpoint:  shared.CheckpointFromConsensus(st.CurrentJustifiedCheckpoint()),
		FinalizedCheckpoint:         shared.CheckpointFromConsensus(st.FinalizedCheckpoint()),
	}, nil
}

type BeaconStateAltair struct {
	GenesisTime                 string                    `json:"genesis_time"`
	GenesisValidatorsRoot       string                    `json:"genesis_validators_root"`
	Slot                        string                    `json:"slot"`
	Fork                        *shared.Fork              `json:"fork"`
	LatestBlockHeader           *shared.BeaconBlockHeader `json:"latest_block_header"`
	BlockRoots                  []string                  `json:"block_roots"`
	StateRoots                  []string                  `json:"state_roots"`
	HistoricalRoots             []string                  `json:"historical_roots"`
	Eth1Data                    *shared.Eth1Data          `json:"eth1_data"`
	Eth1DataVotes               []*shared.Eth1Data        `json:"eth1_data_votes"`
	Eth1DepositIndex            string                    `json:"eth1_deposit_index"`
	Validators                  []*Validator              `json:"validators"`
	Balances                    []string                  `json:"balances"`
	RandaoMixes                 []string                  `json:"randao_mixes"`
	Slashings                   []string                  `json:"slashings"`
	PreviousEpochParticipation  []string                  `json:"previous_epoch_participation"`
	CurrentEpochParticipation   []string                  `json:"current_epoch_participation"`
	JustificationBits           string                    `json:"justification_bits"`
	PreviousJustifiedCheckpoint *shared.Checkpoint        `json:"previous_justified_checkpoint"`
	CurrentJustifiedCheckpoint  *shared.Checkpoint        `json:"current_justified_checkpoint"`
	FinalizedCheckpoint         *shared.Checkpoint        `json:"finalized_checkpoint"`
	InactivityScores            []string                  `json:"inactivity_scores"`
	CurrentSyncCommittee        *SyncCommittee            `json:"current_sync_committee"`
	NextSyncCommittee           *SyncCommittee            `json:"next_sync_committee"`
}

func BeaconStateAltairFromConsensus(st beaconState.BeaconState) (*BeaconStateAltair, error) {
	if st == nil {
		return nil, errors.New("state is empty")
	}

	f, err := shared.ForkFromConsensus(st.Fork())
	if err != nil {
		return nil, err
	}
	srcbr := st.BlockRoots()
	br := make([]string, len(srcbr))
	for i, r := range srcbr {
		br[i] = hexutil.Encode(r)
	}
	srcsr := st.StateRoots()
	sr := make([]string, len(srcsr))
	for i, r := range srcsr {
		sr[i] = hexutil.Encode(r)
	}
	srchr, err := st.HistoricalRoots()
	if err != nil {
		return nil, err
	}
	hr := make([]string, len(srchr))
	for i, r := range srchr {
		hr[i] = hexutil.Encode(r)
	}
	e1d, err := shared.Eth1DataFromConsensus(st.Eth1Data())
	if err != nil {
		return nil, err
	}
	srcvotes := st.Eth1DataVotes()
	votes := make([]*shared.Eth1Data, len(srcvotes))
	for i, e := range srcvotes {
		votes[i], err = shared.Eth1DataFromConsensus(e)
		if err != nil {
			return nil, err
		}
	}
	srcvals := st.Validators()
	vals := make([]*Validator, len(srcvals))
	for i, v := range srcvals {
		vals[i], err = ValidatorFromConsensus(v)
		if err != nil {
			return nil, err
		}
	}
	srcbals := st.Balances()
	bals := make([]string, len(srcbals))
	for i, b := range srcbals {
		bals[i] = fmt.Sprintf("%d", b)
	}
	srcrm := st.RandaoMixes()
	rm := make([]string, len(srcrm))
	for i, m := range srcrm {
		rm[i] = hexutil.Encode(m)
	}
	srcslashings := st.Slashings()
	slashings := make([]string, len(srcslashings))
	for i, s := range srcslashings {
		slashings[i] = fmt.Sprintf("%d", s)
	}
	srcPrevPart, err := st.PreviousEpochParticipation()
	if err != nil {
		return nil, err
	}
	prevPart := make([]string, len(srcPrevPart))
	for i, p := range srcPrevPart {
		prevPart[i] = strconv.FormatUint(uint64(p), 10)
	}
	srcCurrPart, err := st.CurrentEpochParticipation()
	if err != nil {
		return nil, err
	}
	currPart := make([]string, len(srcCurrPart))
	for i, p := range srcCurrPart {
		currPart[i] = strconv.FormatUint(uint64(p), 10)
	}
	srcis, err := st.InactivityScores()
	if err != nil {
		return nil, err
	}
	is := make([]string, len(srcis))
	for i, s := range srcis {
		is[i] = fmt.Sprintf("%d", s)
	}
	srcCurrSc, err := st.CurrentSyncCommittee()
	if err != nil {
		return nil, err
	}
	currSc, err := SyncCommitteeFromConsensus(srcCurrSc)
	if err != nil {
		return nil, err
	}
	srcNextSc, err := st.NextSyncCommittee()
	if err != nil {
		return nil, err
	}
	nextSc, err := SyncCommitteeFromConsensus(srcNextSc)
	if err != nil {
		return nil, err
	}

	return &BeaconStateAltair{
		GenesisTime:                 fmt.Sprintf("%d", st.GenesisTime()),
		GenesisValidatorsRoot:       hexutil.Encode(st.GenesisValidatorsRoot()),
		Slot:                        fmt.Sprintf("%d", st.Slot()),
		Fork:                        f,
		LatestBlockHeader:           shared.BeaconBlockHeaderFromConsensus(st.LatestBlockHeader()),
		BlockRoots:                  br,
		StateRoots:                  sr,
		HistoricalRoots:             hr,
		Eth1Data:                    e1d,
		Eth1DataVotes:               votes,
		Eth1DepositIndex:            fmt.Sprintf("%d", st.Eth1DepositIndex()),
		Validators:                  vals,
		Balances:                    bals,
		RandaoMixes:                 rm,
		Slashings:                   slashings,
		PreviousEpochParticipation:  prevPart,
		CurrentEpochParticipation:   currPart,
		JustificationBits:           hexutil.Encode(st.JustificationBits()),
		PreviousJustifiedCheckpoint: shared.CheckpointFromConsensus(st.PreviousJustifiedCheckpoint()),
		CurrentJustifiedCheckpoint:  shared.CheckpointFromConsensus(st.CurrentJustifiedCheckpoint()),
		FinalizedCheckpoint:         shared.CheckpointFromConsensus(st.FinalizedCheckpoint()),
		InactivityScores:            is,
		CurrentSyncCommittee:        currSc,
		NextSyncCommittee:           nextSc,
	}, nil
}

type BeaconStateBellatrix struct {
	GenesisTime                  string                         `json:"genesis_time"`
	GenesisValidatorsRoot        string                         `json:"genesis_validators_root"`
	Slot                         string                         `json:"slot"`
	Fork                         *shared.Fork                   `json:"fork"`
	LatestBlockHeader            *shared.BeaconBlockHeader      `json:"latest_block_header"`
	BlockRoots                   []string                       `json:"block_roots"`
	StateRoots                   []string                       `json:"state_roots"`
	HistoricalRoots              []string                       `json:"historical_roots"`
	Eth1Data                     *shared.Eth1Data               `json:"eth1_data"`
	Eth1DataVotes                []*shared.Eth1Data             `json:"eth1_data_votes"`
	Eth1DepositIndex             string                         `json:"eth1_deposit_index"`
	Validators                   []*Validator                   `json:"validators"`
	Balances                     []string                       `json:"balances"`
	RandaoMixes                  []string                       `json:"randao_mixes"`
	Slashings                    []string                       `json:"slashings"`
	PreviousEpochParticipation   []string                       `json:"previous_epoch_participation"`
	CurrentEpochParticipation    []string                       `json:"current_epoch_participation"`
	JustificationBits            string                         `json:"justification_bits"`
	PreviousJustifiedCheckpoint  *shared.Checkpoint             `json:"previous_justified_checkpoint"`
	CurrentJustifiedCheckpoint   *shared.Checkpoint             `json:"current_justified_checkpoint"`
	FinalizedCheckpoint          *shared.Checkpoint             `json:"finalized_checkpoint"`
	InactivityScores             []string                       `json:"inactivity_scores"`
	CurrentSyncCommittee         *SyncCommittee                 `json:"current_sync_committee"`
	NextSyncCommittee            *SyncCommittee                 `json:"next_sync_committee"`
	LatestExecutionPayloadHeader *shared.ExecutionPayloadHeader `json:"latest_execution_payload_header"`
}

func BeaconStateBellatrixFromConsensus(st beaconState.BeaconState) (*BeaconStateBellatrix, error) {
	if st == nil {
		return nil, errors.New("state is empty")
	}

	f, err := shared.ForkFromConsensus(st.Fork())
	if err != nil {
		return nil, err
	}
	srcbr := st.BlockRoots()
	br := make([]string, len(srcbr))
	for i, r := range srcbr {
		br[i] = hexutil.Encode(r)
	}
	srcsr := st.StateRoots()
	sr := make([]string, len(srcsr))
	for i, r := range srcsr {
		sr[i] = hexutil.Encode(r)
	}
	srchr, err := st.HistoricalRoots()
	if err != nil {
		return nil, err
	}
	hr := make([]string, len(srchr))
	for i, r := range srchr {
		hr[i] = hexutil.Encode(r)
	}
	e1d, err := shared.Eth1DataFromConsensus(st.Eth1Data())
	if err != nil {
		return nil, err
	}
	srcvotes := st.Eth1DataVotes()
	votes := make([]*shared.Eth1Data, len(srcvotes))
	for i, e := range srcvotes {
		votes[i], err = shared.Eth1DataFromConsensus(e)
		if err != nil {
			return nil, err
		}
	}
	srcvals := st.Validators()
	vals := make([]*Validator, len(srcvals))
	for i, v := range srcvals {
		vals[i], err = ValidatorFromConsensus(v)
		if err != nil {
			return nil, err
		}
	}
	srcbals := st.Balances()
	bals := make([]string, len(srcbals))
	for i, b := range srcbals {
		bals[i] = fmt.Sprintf("%d", b)
	}
	srcrm := st.RandaoMixes()
	rm := make([]string, len(srcrm))
	for i, m := range srcrm {
		rm[i] = hexutil.Encode(m)
	}
	srcslashings := st.Slashings()
	slashings := make([]string, len(srcslashings))
	for i, s := range srcslashings {
		slashings[i] = fmt.Sprintf("%d", s)
	}
	srcPrevPart, err := st.PreviousEpochParticipation()
	if err != nil {
		return nil, err
	}
	prevPart := make([]string, len(srcPrevPart))
	for i, p := range srcPrevPart {
		prevPart[i] = strconv.FormatUint(uint64(p), 10)
	}
	srcCurrPart, err := st.CurrentEpochParticipation()
	if err != nil {
		return nil, err
	}
	currPart := make([]string, len(srcCurrPart))
	for i, p := range srcCurrPart {
		currPart[i] = strconv.FormatUint(uint64(p), 10)
	}
	srcis, err := st.InactivityScores()
	if err != nil {
		return nil, err
	}
	is := make([]string, len(srcis))
	for i, s := range srcis {
		is[i] = fmt.Sprintf("%d", s)
	}
	srcCurrSc, err := st.CurrentSyncCommittee()
	if err != nil {
		return nil, err
	}
	currSc, err := SyncCommitteeFromConsensus(srcCurrSc)
	if err != nil {
		return nil, err
	}
	srcNextSc, err := st.NextSyncCommittee()
	if err != nil {
		return nil, err
	}
	nextSc, err := SyncCommitteeFromConsensus(srcNextSc)
	if err != nil {
		return nil, err
	}
	execData, err := st.LatestExecutionPayloadHeader()
	if err != nil {
		return nil, err
	}
	srcPayload, ok := execData.Proto().(*enginev1.ExecutionPayloadHeader)
	if !ok {
		return nil, errPayloadHeaderNotFound
	}
	payload, err := shared.ExecutionPayloadHeaderFromConsensus(srcPayload)
	if err != nil {
		return nil, err
	}

	return &BeaconStateBellatrix{
		GenesisTime:                  fmt.Sprintf("%d", st.GenesisTime()),
		GenesisValidatorsRoot:        hexutil.Encode(st.GenesisValidatorsRoot()),
		Slot:                         fmt.Sprintf("%d", st.Slot()),
		Fork:                         f,
		LatestBlockHeader:            shared.BeaconBlockHeaderFromConsensus(st.LatestBlockHeader()),
		BlockRoots:                   br,
		StateRoots:                   sr,
		HistoricalRoots:              hr,
		Eth1Data:                     e1d,
		Eth1DataVotes:                votes,
		Eth1DepositIndex:             fmt.Sprintf("%d", st.Eth1DepositIndex()),
		Validators:                   vals,
		Balances:                     bals,
		RandaoMixes:                  rm,
		Slashings:                    slashings,
		PreviousEpochParticipation:   prevPart,
		CurrentEpochParticipation:    currPart,
		JustificationBits:            hexutil.Encode(st.JustificationBits()),
		PreviousJustifiedCheckpoint:  shared.CheckpointFromConsensus(st.PreviousJustifiedCheckpoint()),
		CurrentJustifiedCheckpoint:   shared.CheckpointFromConsensus(st.CurrentJustifiedCheckpoint()),
		FinalizedCheckpoint:          shared.CheckpointFromConsensus(st.FinalizedCheckpoint()),
		InactivityScores:             is,
		CurrentSyncCommittee:         currSc,
		NextSyncCommittee:            nextSc,
		LatestExecutionPayloadHeader: payload,
	}, nil
}

type BeaconStateCapella struct {
	GenesisTime                  string                                `json:"genesis_time"`
	GenesisValidatorsRoot        string                                `json:"genesis_validators_root"`
	Slot                         string                                `json:"slot"`
	Fork                         *shared.Fork                          `json:"fork"`
	LatestBlockHeader            *shared.BeaconBlockHeader             `json:"latest_block_header"`
	BlockRoots                   []string                              `json:"block_roots"`
	StateRoots                   []string                              `json:"state_roots"`
	HistoricalRoots              []string                              `json:"historical_roots"`
	Eth1Data                     *shared.Eth1Data                      `json:"eth1_data"`
	Eth1DataVotes                []*shared.Eth1Data                    `json:"eth1_data_votes"`
	Eth1DepositIndex             string                                `json:"eth1_deposit_index"`
	Validators                   []*Validator                          `json:"validators"`
	Balances                     []string                              `json:"balances"`
	RandaoMixes                  []string                              `json:"randao_mixes"`
	Slashings                    []string                              `json:"slashings"`
	PreviousEpochParticipation   []string                              `json:"previous_epoch_participation"`
	CurrentEpochParticipation    []string                              `json:"current_epoch_participation"`
	JustificationBits            string                                `json:"justification_bits"`
	PreviousJustifiedCheckpoint  *shared.Checkpoint                    `json:"previous_justified_checkpoint"`
	CurrentJustifiedCheckpoint   *shared.Checkpoint                    `json:"current_justified_checkpoint"`
	FinalizedCheckpoint          *shared.Checkpoint                    `json:"finalized_checkpoint"`
	InactivityScores             []string                              `json:"inactivity_scores"`
	CurrentSyncCommittee         *SyncCommittee                        `json:"current_sync_committee"`
	NextSyncCommittee            *SyncCommittee                        `json:"next_sync_committee"`
	LatestExecutionPayloadHeader *shared.ExecutionPayloadHeaderCapella `json:"latest_execution_payload_header"`
	NextWithdrawalIndex          string                                `json:"next_withdrawal_index"`
	NextWithdrawalValidatorIndex string                                `json:"next_withdrawal_validator_index"`
	HistoricalSummaries          []*HistoricalSummary                  `json:"historical_summaries"`
}

func BeaconStateCapellaFromConsensus(st beaconState.BeaconState) (*BeaconStateCapella, error) {
	if st == nil {
		return nil, errors.New("state is empty")
	}

	f, err := shared.ForkFromConsensus(st.Fork())
	if err != nil {
		return nil, err
	}
	srcbr := st.BlockRoots()
	br := make([]string, len(srcbr))
	for i, r := range srcbr {
		br[i] = hexutil.Encode(r)
	}
	srcsr := st.StateRoots()
	sr := make([]string, len(srcsr))
	for i, r := range srcsr {
		sr[i] = hexutil.Encode(r)
	}
	srchr, err := st.HistoricalRoots()
	if err != nil {
		return nil, err
	}
	hr := make([]string, len(srchr))
	for i, r := range srchr {
		hr[i] = hexutil.Encode(r)
	}
	e1d, err := shared.Eth1DataFromConsensus(st.Eth1Data())
	if err != nil {
		return nil, err
	}
	srcvotes := st.Eth1DataVotes()
	votes := make([]*shared.Eth1Data, len(srcvotes))
	for i, e := range srcvotes {
		votes[i], err = shared.Eth1DataFromConsensus(e)
		if err != nil {
			return nil, err
		}
	}
	srcvals := st.Validators()
	vals := make([]*Validator, len(srcvals))
	for i, v := range srcvals {
		vals[i], err = ValidatorFromConsensus(v)
		if err != nil {
			return nil, err
		}
	}
	srcbals := st.Balances()
	bals := make([]string, len(srcbals))
	for i, b := range srcbals {
		bals[i] = fmt.Sprintf("%d", b)
	}
	srcrm := st.RandaoMixes()
	rm := make([]string, len(srcrm))
	for i, m := range srcrm {
		rm[i] = hexutil.Encode(m)
	}
	srcslashings := st.Slashings()
	slashings := make([]string, len(srcslashings))
	for i, s := range srcslashings {
		slashings[i] = fmt.Sprintf("%d", s)
	}
	srcPrevPart, err := st.PreviousEpochParticipation()
	if err != nil {
		return nil, err
	}
	prevPart := make([]string, len(srcPrevPart))
	for i, p := range srcPrevPart {
		prevPart[i] = strconv.FormatUint(uint64(p), 10)
	}
	srcCurrPart, err := st.CurrentEpochParticipation()
	if err != nil {
		return nil, err
	}
	currPart := make([]string, len(srcCurrPart))
	for i, p := range srcCurrPart {
		currPart[i] = strconv.FormatUint(uint64(p), 10)
	}
	srcis, err := st.InactivityScores()
	if err != nil {
		return nil, err
	}
	is := make([]string, len(srcis))
	for i, s := range srcis {
		is[i] = fmt.Sprintf("%d", s)
	}
	srcCurrSc, err := st.CurrentSyncCommittee()
	if err != nil {
		return nil, err
	}
	currSc, err := SyncCommitteeFromConsensus(srcCurrSc)
	if err != nil {
		return nil, err
	}
	srcNextSc, err := st.NextSyncCommittee()
	if err != nil {
		return nil, err
	}
	nextSc, err := SyncCommitteeFromConsensus(srcNextSc)
	if err != nil {
		return nil, err
	}
	execData, err := st.LatestExecutionPayloadHeader()
	if err != nil {
		return nil, err
	}
	srcPayload, ok := execData.Proto().(*enginev1.ExecutionPayloadHeaderCapella)
	if !ok {
		return nil, errPayloadHeaderNotFound
	}
	payload, err := shared.ExecutionPayloadHeaderCapellaFromConsensus(srcPayload)
	if err != nil {
		return nil, err
	}
	srchs, err := st.HistoricalSummaries()
	if err != nil {
		return nil, err
	}
	hs := make([]*HistoricalSummary, len(srchs))
	for i, s := range srchs {
		hs[i], err = HistoricalSummaryFromConsensus(s)
		if err != nil {
			return nil, err
		}
	}
	nwi, err := st.NextWithdrawalIndex()
	if err != nil {
		return nil, err
	}
	nwvi, err := st.NextWithdrawalValidatorIndex()
	if err != nil {
		return nil, err
	}

	return &BeaconStateCapella{
		GenesisTime:                  fmt.Sprintf("%d", st.GenesisTime()),
		GenesisValidatorsRoot:        hexutil.Encode(st.GenesisValidatorsRoot()),
		Slot:                         fmt.Sprintf("%d", st.Slot()),
		Fork:                         f,
		LatestBlockHeader:            shared.BeaconBlockHeaderFromConsensus(st.LatestBlockHeader()),
		BlockRoots:                   br,
		StateRoots:                   sr,
		HistoricalRoots:              hr,
		Eth1Data:                     e1d,
		Eth1DataVotes:                votes,
		Eth1DepositIndex:             fmt.Sprintf("%d", st.Eth1DepositIndex()),
		Validators:                   vals,
		Balances:                     bals,
		RandaoMixes:                  rm,
		Slashings:                    slashings,
		PreviousEpochParticipation:   prevPart,
		CurrentEpochParticipation:    currPart,
		JustificationBits:            hexutil.Encode(st.JustificationBits()),
		PreviousJustifiedCheckpoint:  shared.CheckpointFromConsensus(st.PreviousJustifiedCheckpoint()),
		CurrentJustifiedCheckpoint:   shared.CheckpointFromConsensus(st.CurrentJustifiedCheckpoint()),
		FinalizedCheckpoint:          shared.CheckpointFromConsensus(st.FinalizedCheckpoint()),
		InactivityScores:             is,
		CurrentSyncCommittee:         currSc,
		NextSyncCommittee:            nextSc,
		LatestExecutionPayloadHeader: payload,
		NextWithdrawalIndex:          fmt.Sprintf("%d", nwi),
		NextWithdrawalValidatorIndex: fmt.Sprintf("%d", nwvi),
		HistoricalSummaries:          hs,
	}, nil
}

type BeaconStateDeneb struct {
	GenesisTime                  string                              `json:"genesis_time"`
	GenesisValidatorsRoot        string                              `json:"genesis_validators_root"`
	Slot                         string                              `json:"slot"`
	Fork                         *shared.Fork                        `json:"fork"`
	LatestBlockHeader            *shared.BeaconBlockHeader           `json:"latest_block_header"`
	BlockRoots                   []string                            `json:"block_roots"`
	StateRoots                   []string                            `json:"state_roots"`
	HistoricalRoots              []string                            `json:"historical_roots"`
	Eth1Data                     *shared.Eth1Data                    `json:"eth1_data"`
	Eth1DataVotes                []*shared.Eth1Data                  `json:"eth1_data_votes"`
	Eth1DepositIndex             string                              `json:"eth1_deposit_index"`
	Validators                   []*Validator                        `json:"validators"`
	Balances                     []string                            `json:"balances"`
	RandaoMixes                  []string                            `json:"randao_mixes"`
	Slashings                    []string                            `json:"slashings"`
	PreviousEpochParticipation   []string                            `json:"previous_epoch_participation"`
	CurrentEpochParticipation    []string                            `json:"current_epoch_participation"`
	JustificationBits            string                              `json:"justification_bits"`
	PreviousJustifiedCheckpoint  *shared.Checkpoint                  `json:"previous_justified_checkpoint"`
	CurrentJustifiedCheckpoint   *shared.Checkpoint                  `json:"current_justified_checkpoint"`
	FinalizedCheckpoint          *shared.Checkpoint                  `json:"finalized_checkpoint"`
	InactivityScores             []string                            `json:"inactivity_scores"`
	CurrentSyncCommittee         *SyncCommittee                      `json:"current_sync_committee"`
	NextSyncCommittee            *SyncCommittee                      `json:"next_sync_committee"`
	LatestExecutionPayloadHeader *shared.ExecutionPayloadHeaderDeneb `json:"latest_execution_payload_header"`
	NextWithdrawalIndex          string                              `json:"next_withdrawal_index"`
	NextWithdrawalValidatorIndex string                              `json:"next_withdrawal_validator_index"`
	HistoricalSummaries          []*HistoricalSummary                `json:"historical_summaries"`
}

func BeaconStateDenebFromConsensus(st beaconState.BeaconState) (*BeaconStateDeneb, error) {
	if st == nil {
		return nil, errors.New("state is empty")
	}

	f, err := shared.ForkFromConsensus(st.Fork())
	if err != nil {
		return nil, err
	}
	srcbr := st.BlockRoots()
	br := make([]string, len(srcbr))
	for i, r := range srcbr {
		br[i] = hexutil.Encode(r)
	}
	srcsr := st.StateRoots()
	sr := make([]string, len(srcsr))
	for i, r := range srcsr {
		sr[i] = hexutil.Encode(r)
	}
	srchr, err := st.HistoricalRoots()
	if err != nil {
		return nil, err
	}
	hr := make([]string, len(srchr))
	for i, r := range srchr {
		hr[i] = hexutil.Encode(r)
	}
	e1d, err := shared.Eth1DataFromConsensus(st.Eth1Data())
	if err != nil {
		return nil, err
	}
	srcvotes := st.Eth1DataVotes()
	votes := make([]*shared.Eth1Data, len(srcvotes))
	for i, e := range srcvotes {
		votes[i], err = shared.Eth1DataFromConsensus(e)
		if err != nil {
			return nil, err
		}
	}
	srcvals := st.Validators()
	vals := make([]*Validator, len(srcvals))
	for i, v := range srcvals {
		vals[i], err = ValidatorFromConsensus(v)
		if err != nil {
			return nil, err
		}
	}
	srcbals := st.Balances()
	bals := make([]string, len(srcbals))
	for i, b := range srcbals {
		bals[i] = fmt.Sprintf("%d", b)
	}
	srcrm := st.RandaoMixes()
	rm := make([]string, len(srcrm))
	for i, m := range srcrm {
		rm[i] = hexutil.Encode(m)
	}
	srcslashings := st.Slashings()
	slashings := make([]string, len(srcslashings))
	for i, s := range srcslashings {
		slashings[i] = fmt.Sprintf("%d", s)
	}
	srcPrevPart, err := st.PreviousEpochParticipation()
	if err != nil {
		return nil, err
	}
	prevPart := make([]string, len(srcPrevPart))
	for i, p := range srcPrevPart {
		prevPart[i] = strconv.FormatUint(uint64(p), 10)
	}
	srcCurrPart, err := st.CurrentEpochParticipation()
	if err != nil {
		return nil, err
	}
	currPart := make([]string, len(srcCurrPart))
	for i, p := range srcCurrPart {
		currPart[i] = strconv.FormatUint(uint64(p), 10)
	}
	srcis, err := st.InactivityScores()
	if err != nil {
		return nil, err
	}
	is := make([]string, len(srcis))
	for i, s := range srcis {
		is[i] = fmt.Sprintf("%d", s)
	}
	srcCurrSc, err := st.CurrentSyncCommittee()
	if err != nil {
		return nil, err
	}
	currSc, err := SyncCommitteeFromConsensus(srcCurrSc)
	if err != nil {
		return nil, err
	}
	srcNextSc, err := st.NextSyncCommittee()
	if err != nil {
		return nil, err
	}
	nextSc, err := SyncCommitteeFromConsensus(srcNextSc)
	if err != nil {
		return nil, err
	}
	execData, err := st.LatestExecutionPayloadHeader()
	if err != nil {
		return nil, err
	}
	srcPayload, ok := execData.Proto().(*enginev1.ExecutionPayloadHeaderDeneb)
	if !ok {
		return nil, errPayloadHeaderNotFound
	}
	payload, err := shared.ExecutionPayloadHeaderDenebFromConsensus(srcPayload)
	if err != nil {
		return nil, err
	}
	srchs, err := st.HistoricalSummaries()
	if err != nil {
		return nil, err
	}
	hs := make([]*HistoricalSummary, len(srchs))
	for i, s := range srchs {
		hs[i], err = HistoricalSummaryFromConsensus(s)
		if err != nil {
			return nil, err
		}
	}
	nwi, err := st.NextWithdrawalIndex()
	if err != nil {
		return nil, err
	}
	nwvi, err := st.NextWithdrawalValidatorIndex()
	if err != nil {
		return nil, err
	}

	return &BeaconStateDeneb{
		GenesisTime:                  fmt.Sprintf("%d", st.GenesisTime()),
		GenesisValidatorsRoot:        hexutil.Encode(st.GenesisValidatorsRoot()),
		Slot:                         fmt.Sprintf("%d", st.Slot()),
		Fork:                         f,
		LatestBlockHeader:            shared.BeaconBlockHeaderFromConsensus(st.LatestBlockHeader()),
		BlockRoots:                   br,
		StateRoots:                   sr,
		HistoricalRoots:              hr,
		Eth1Data:                     e1d,
		Eth1DataVotes:                votes,
		Eth1DepositIndex:             fmt.Sprintf("%d", st.Eth1DepositIndex()),
		Validators:                   vals,
		Balances:                     bals,
		RandaoMixes:                  rm,
		Slashings:                    slashings,
		PreviousEpochParticipation:   prevPart,
		CurrentEpochParticipation:    currPart,
		JustificationBits:            hexutil.Encode(st.JustificationBits()),
		PreviousJustifiedCheckpoint:  shared.CheckpointFromConsensus(st.PreviousJustifiedCheckpoint()),
		CurrentJustifiedCheckpoint:   shared.CheckpointFromConsensus(st.CurrentJustifiedCheckpoint()),
		FinalizedCheckpoint:          shared.CheckpointFromConsensus(st.FinalizedCheckpoint()),
		InactivityScores:             is,
		CurrentSyncCommittee:         currSc,
		NextSyncCommittee:            nextSc,
		LatestExecutionPayloadHeader: payload,
		NextWithdrawalIndex:          fmt.Sprintf("%d", nwi),
		NextWithdrawalValidatorIndex: fmt.Sprintf("%d", nwvi),
		HistoricalSummaries:          hs,
	}, nil
}

type Validator struct {
	PublicKey                  string `json:"pubkey"`
	WithdrawalCredentials      string `json:"withdrawal_credentials"`
	EffectiveBalance           string `json:"effective_balance"`
	Slashed                    bool   `json:"slashed"`
	ActivationEligibilityEpoch string `json:"activation_eligibility_epoch"`
	ActivationEpoch            string `json:"activation_epoch"`
	ExitEpoch                  string `json:"exit_epoch"`
	WithdrawableEpoch          string `json:"withdrawable_epoch"`
}

func ValidatorFromConsensus(v *eth.Validator) (*Validator, error) {
	if v == nil {
		return nil, errors.New("validator is empty")
	}

	return &Validator{
		PublicKey:                  hexutil.Encode(v.PublicKey),
		WithdrawalCredentials:      hexutil.Encode(v.WithdrawalCredentials),
		EffectiveBalance:           fmt.Sprintf("%d", v.EffectiveBalance),
		Slashed:                    v.Slashed,
		ActivationEligibilityEpoch: fmt.Sprintf("%d", v.ActivationEligibilityEpoch),
		ActivationEpoch:            fmt.Sprintf("%d", v.ActivationEpoch),
		ExitEpoch:                  fmt.Sprintf("%d", v.ExitEpoch),
		WithdrawableEpoch:          fmt.Sprintf("%d", v.WithdrawableEpoch),
	}, nil
}

type PendingAttestation struct {
	AggregationBits string                  `json:"aggregation_bits"`
	Data            *shared.AttestationData `json:"data"`
	InclusionDelay  string                  `json:"inclusion_delay"`
	ProposerIndex   string                  `json:"proposer_index"`
}

func PendingAttestationFromConsensus(a *eth.PendingAttestation) (*PendingAttestation, error) {
	if a == nil {
		return nil, errors.New("pending attestation is empty")
	}

	return &PendingAttestation{
		AggregationBits: hexutil.Encode(a.AggregationBits),
		Data:            nil,
		InclusionDelay:  fmt.Sprintf("%d", a.InclusionDelay),
		ProposerIndex:   fmt.Sprintf("%d", a.ProposerIndex),
	}, nil
}

type SyncCommittee struct {
	Pubkeys         []string `json:"pubkeys"`
	AggregatePubkey string   `json:"aggregate_pubkey"`
}

func SyncCommitteeFromConsensus(sc *eth.SyncCommittee) (*SyncCommittee, error) {
	if sc == nil {
		return nil, errors.New("sync committee is empty")
	}

	pubkeys := make([]string, len(sc.Pubkeys))
	for i, p := range sc.Pubkeys {
		pubkeys[i] = hexutil.Encode(p)
	}
	return &SyncCommittee{
		Pubkeys:         pubkeys,
		AggregatePubkey: hexutil.Encode(sc.AggregatePubkey),
	}, nil
}

type HistoricalSummary struct {
	BlockSummaryRoot string `json:"block_summary_root"`
	StateSummaryRoot string `json:"state_summary_root"`
}

func HistoricalSummaryFromConsensus(summary *eth.HistoricalSummary) (*HistoricalSummary, error) {
	if summary == nil {
		return nil, errors.New("historical summary is empty")
	}

	return &HistoricalSummary{
		BlockSummaryRoot: hexutil.Encode(summary.BlockSummaryRoot),
		StateSummaryRoot: hexutil.Encode(summary.StateSummaryRoot),
	}, nil
}
