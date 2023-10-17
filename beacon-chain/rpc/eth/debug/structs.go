package debug

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

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

func BeaconStateFromConsensus(st *eth.BeaconState) (*BeaconState, error) {
	if st == nil {
		return nil, errors.New("state is empty")
	}

	f, err := shared.ForkFromConsensus(st.Fork)
	if err != nil {
		return nil, err
	}
	br := make([]string, len(st.BlockRoots))
	for i, r := range st.BlockRoots {
		br[i] = hexutil.Encode(r)
	}
	sr := make([]string, len(st.StateRoots))
	for i, r := range st.StateRoots {
		sr[i] = hexutil.Encode(r)
	}
	hr := make([]string, len(st.HistoricalRoots))
	for i, r := range st.HistoricalRoots {
		hr[i] = hexutil.Encode(r)
	}
	e1d, err := shared.Eth1DataFromConsensus(st.Eth1Data)
	if err != nil {
		return nil, err
	}
	e1dVotes := make([]*shared.Eth1Data, len(st.Eth1DataVotes))
	for i, e := range st.Eth1DataVotes {
		e1dVotes[i], err = shared.Eth1DataFromConsensus(e)
		if err != nil {
			return nil, err
		}
	}
	vals := make([]*Validator, len(st.Validators))
	for i, v := range st.Validators {
		vals[i], err = ValidatorFromConsensus(v)
		if err != nil {
			return nil, err
		}
	}
	bals := make([]string, len(st.Balances))
	for i, b := range st.Balances {
		bals[i] = fmt.Sprintf("%d", b)
	}
	rm := make([]string, len(st.RandaoMixes))
	for i, m := range st.RandaoMixes {
		rm[i] = hexutil.Encode(m)
	}
	slashings := make([]string, len(st.Slashings))
	for i, s := range st.Slashings {
		slashings[i] = fmt.Sprintf("%d", s)
	}
	prevAtts := make([]*PendingAttestation, len(st.PreviousEpochAttestations))
	for i, a := range st.PreviousEpochAttestations {
		prevAtts[i], err = PendingAttestationFromConsensus(a)
		if err != nil {
			return nil, err
		}
	}
	currAtts := make([]*PendingAttestation, len(st.CurrentEpochAttestations))
	for i, a := range st.CurrentEpochAttestations {
		currAtts[i], err = PendingAttestationFromConsensus(a)
		if err != nil {
			return nil, err
		}
	}

	return &BeaconState{
		GenesisTime:                 fmt.Sprintf("%d", st.GenesisTime),
		GenesisValidatorsRoot:       hexutil.Encode(st.GenesisValidatorsRoot),
		Slot:                        fmt.Sprintf("%d", st.Slot),
		Fork:                        f,
		LatestBlockHeader:           shared.BeaconBlockHeaderFromConsensus(st.LatestBlockHeader),
		BlockRoots:                  br,
		StateRoots:                  sr,
		HistoricalRoots:             hr,
		Eth1Data:                    e1d,
		Eth1DataVotes:               e1dVotes,
		Eth1DepositIndex:            fmt.Sprintf("%d", st.Eth1DepositIndex),
		Validators:                  vals,
		Balances:                    bals,
		RandaoMixes:                 rm,
		Slashings:                   slashings,
		PreviousEpochAttestations:   prevAtts,
		CurrentEpochAttestations:    currAtts,
		JustificationBits:           hexutil.Encode(st.JustificationBits),
		PreviousJustifiedCheckpoint: shared.CheckpointFromConsensus(st.PreviousJustifiedCheckpoint),
		CurrentJustifiedCheckpoint:  shared.CheckpointFromConsensus(st.CurrentJustifiedCheckpoint),
		FinalizedCheckpoint:         shared.CheckpointFromConsensus(st.FinalizedCheckpoint),
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
	PreviousEpochParticipation  EpochParticipation        `json:"previous_epoch_participation"`
	CurrentEpochParticipation   EpochParticipation        `json:"current_epoch_participation"`
	JustificationBits           string                    `json:"justification_bits"`
	PreviousJustifiedCheckpoint *shared.Checkpoint        `json:"previous_justified_checkpoint"`
	CurrentJustifiedCheckpoint  *shared.Checkpoint        `json:"current_justified_checkpoint"`
	FinalizedCheckpoint         *shared.Checkpoint        `json:"finalized_checkpoint"`
	InactivityScores            []string                  `json:"inactivity_scores"`
	CurrentSyncCommittee        *SyncCommittee            `json:"current_sync_committee"`
	NextSyncCommittee           *SyncCommittee            `json:"next_sync_committee"`
}

func BeaconStateAltairFromConsensus(st *eth.BeaconStateAltair) (*BeaconStateAltair, error) {
	if st == nil {
		return nil, errors.New("state is empty")
	}

	f, err := shared.ForkFromConsensus(st.Fork)
	if err != nil {
		return nil, err
	}
	br := make([]string, len(st.BlockRoots))
	for i, r := range st.BlockRoots {
		br[i] = hexutil.Encode(r)
	}
	sr := make([]string, len(st.StateRoots))
	for i, r := range st.StateRoots {
		sr[i] = hexutil.Encode(r)
	}
	hr := make([]string, len(st.HistoricalRoots))
	for i, r := range st.HistoricalRoots {
		hr[i] = hexutil.Encode(r)
	}
	e1d, err := shared.Eth1DataFromConsensus(st.Eth1Data)
	if err != nil {
		return nil, err
	}
	e1dVotes := make([]*shared.Eth1Data, len(st.Eth1DataVotes))
	for i, e := range st.Eth1DataVotes {
		e1dVotes[i], err = shared.Eth1DataFromConsensus(e)
		if err != nil {
			return nil, err
		}
	}
	vals := make([]*Validator, len(st.Validators))
	for i, v := range st.Validators {
		vals[i], err = ValidatorFromConsensus(v)
		if err != nil {
			return nil, err
		}
	}
	bals := make([]string, len(st.Balances))
	for i, b := range st.Balances {
		bals[i] = fmt.Sprintf("%d", b)
	}
	rm := make([]string, len(st.RandaoMixes))
	for i, m := range st.RandaoMixes {
		rm[i] = hexutil.Encode(m)
	}
	slashings := make([]string, len(st.Slashings))
	for i, s := range st.Slashings {
		slashings[i] = fmt.Sprintf("%d", s)
	}
	is := make([]string, len(st.InactivityScores))
	for i, s := range st.InactivityScores {
		is[i] = fmt.Sprintf("%d", s)
	}
	currSc, err := SyncCommitteeFromConsensus(st.CurrentSyncCommittee)
	if err != nil {
		return nil, err
	}
	nextSc, err := SyncCommitteeFromConsensus(st.NextSyncCommittee)
	if err != nil {
		return nil, err
	}

	return &BeaconStateAltair{
		GenesisTime:                 fmt.Sprintf("%d", st.GenesisTime),
		GenesisValidatorsRoot:       hexutil.Encode(st.GenesisValidatorsRoot),
		Slot:                        fmt.Sprintf("%d", st.Slot),
		Fork:                        f,
		LatestBlockHeader:           shared.BeaconBlockHeaderFromConsensus(st.LatestBlockHeader),
		BlockRoots:                  br,
		StateRoots:                  sr,
		HistoricalRoots:             hr,
		Eth1Data:                    e1d,
		Eth1DataVotes:               e1dVotes,
		Eth1DepositIndex:            fmt.Sprintf("%d", st.Eth1DepositIndex),
		Validators:                  vals,
		Balances:                    bals,
		RandaoMixes:                 rm,
		Slashings:                   slashings,
		PreviousEpochParticipation:  ,
		CurrentEpochParticipation:   nil,
		JustificationBits:           hexutil.Encode(st.JustificationBits),
		PreviousJustifiedCheckpoint: shared.CheckpointFromConsensus(st.PreviousJustifiedCheckpoint),
		CurrentJustifiedCheckpoint:  shared.CheckpointFromConsensus(st.CurrentJustifiedCheckpoint),
		FinalizedCheckpoint:         shared.CheckpointFromConsensus(st.FinalizedCheckpoint),
		InactivityScores:            is,
		CurrentSyncCommittee:        currSc,
		NextSyncCommittee:           nextSc,
	}, nil
}

type BeaconStateBellatrix struct {
	GenesisTime                  string                         `json:"genesis_time"`
	GenesisValidatorsRoot        string                         `json:"genesis_validators_root" hex:"true"`
	Slot                         string                         `json:"slot"`
	Fork                         *shared.Fork                   `json:"fork"`
	LatestBlockHeader            *shared.BeaconBlockHeader      `json:"latest_block_header"`
	BlockRoots                   []string                       `json:"block_roots" hex:"true"`
	StateRoots                   []string                       `json:"state_roots" hex:"true"`
	HistoricalRoots              []string                       `json:"historical_roots" hex:"true"`
	Eth1Data                     *shared.Eth1Data               `json:"eth1_data"`
	Eth1DataVotes                []*shared.Eth1Data             `json:"eth1_data_votes"`
	Eth1DepositIndex             string                         `json:"eth1_deposit_index"`
	Validators                   []*Validator                   `json:"validators"`
	Balances                     []string                       `json:"balances"`
	RandaoMixes                  []string                       `json:"randao_mixes" hex:"true"`
	Slashings                    []string                       `json:"slashings"`
	PreviousEpochParticipation   EpochParticipation             `json:"previous_epoch_participation"`
	CurrentEpochParticipation    EpochParticipation             `json:"current_epoch_participation"`
	JustificationBits            string                         `json:"justification_bits" hex:"true"`
	PreviousJustifiedCheckpoint  *shared.Checkpoint             `json:"previous_justified_checkpoint"`
	CurrentJustifiedCheckpoint   *shared.Checkpoint             `json:"current_justified_checkpoint"`
	FinalizedCheckpoint          *shared.Checkpoint             `json:"finalized_checkpoint"`
	InactivityScores             []string                       `json:"inactivity_scores"`
	CurrentSyncCommittee         *SyncCommittee                 `json:"current_sync_committee"`
	NextSyncCommittee            *SyncCommittee                 `json:"next_sync_committee"`
	LatestExecutionPayloadHeader *shared.ExecutionPayloadHeader `json:"latest_execution_payload_header"`
}

type BeaconStateCapellaJson struct {
	GenesisTime                  string                                `json:"genesis_time"`
	GenesisValidatorsRoot        string                                `json:"genesis_validators_root" hex:"true"`
	Slot                         string                                `json:"slot"`
	Fork                         *shared.Fork                          `json:"fork"`
	LatestBlockHeader            *shared.BeaconBlockHeader             `json:"latest_block_header"`
	BlockRoots                   []string                              `json:"block_roots" hex:"true"`
	StateRoots                   []string                              `json:"state_roots" hex:"true"`
	HistoricalRoots              []string                              `json:"historical_roots" hex:"true"`
	Eth1Data                     *shared.Eth1Data                      `json:"eth1_data"`
	Eth1DataVotes                []*shared.Eth1Data                    `json:"eth1_data_votes"`
	Eth1DepositIndex             string                                `json:"eth1_deposit_index"`
	Validators                   []*Validator                          `json:"validators"`
	Balances                     []string                              `json:"balances"`
	RandaoMixes                  []string                              `json:"randao_mixes" hex:"true"`
	Slashings                    []string                              `json:"slashings"`
	PreviousEpochParticipation   EpochParticipation                    `json:"previous_epoch_participation"`
	CurrentEpochParticipation    EpochParticipation                    `json:"current_epoch_participation"`
	JustificationBits            string                                `json:"justification_bits" hex:"true"`
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

type BeaconStateDenebJson struct {
	GenesisTime                  string                              `json:"genesis_time"`
	GenesisValidatorsRoot        string                              `json:"genesis_validators_root" hex:"true"`
	Slot                         string                              `json:"slot"`
	Fork                         *shared.Fork                        `json:"fork"`
	LatestBlockHeader            *shared.BeaconBlockHeader           `json:"latest_block_header"`
	BlockRoots                   []string                            `json:"block_roots" hex:"true"`
	StateRoots                   []string                            `json:"state_roots" hex:"true"`
	HistoricalRoots              []string                            `json:"historical_roots" hex:"true"`
	Eth1Data                     *shared.Eth1Data                    `json:"eth1_data"`
	Eth1DataVotes                []*shared.Eth1Data                  `json:"eth1_data_votes"`
	Eth1DepositIndex             string                              `json:"eth1_deposit_index"`
	Validators                   []*Validator                        `json:"validators"`
	Balances                     []string                            `json:"balances"`
	RandaoMixes                  []string                            `json:"randao_mixes" hex:"true"`
	Slashings                    []string                            `json:"slashings"`
	PreviousEpochParticipation   EpochParticipation                  `json:"previous_epoch_participation"`
	CurrentEpochParticipation    EpochParticipation                  `json:"current_epoch_participation"`
	JustificationBits            string                              `json:"justification_bits" hex:"true"`
	PreviousJustifiedCheckpoint  *shared.Checkpoint                  `json:"previous_justified_checkpoint"`
	CurrentJustifiedCheckpoint   *shared.Checkpoint                  `json:"current_justified_checkpoint"`
	FinalizedCheckpoint          *shared.Checkpoint                  `json:"finalized_checkpoint"`
	InactivityScores             []string                            `json:"inactivity_scores"`
	CurrentSyncCommittee         *SyncCommittee                      `json:"current_sync_committee"`
	NextSyncCommittee            *SyncCommittee                      `json:"next_sync_committee"`
	LatestExecutionPayloadHeader *shared.ExecutionPayloadHeaderDeneb `json:"latest_execution_payload_header"` // new in deneb
	NextWithdrawalIndex          string                              `json:"next_withdrawal_index"`
	NextWithdrawalValidatorIndex string                              `json:"next_withdrawal_validator_index"`
	HistoricalSummaries          []*HistoricalSummary                `json:"historical_summaries"`
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

// EpochParticipation represents participation of validators in their duties.
type EpochParticipation []string

func (p *EpochParticipation) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}
	if len(b) < 2 {
		return errors.New("epoch participation length must be at least 2")
	}
	if b[0] != '"' || b[len(b)-1] != '"' {
		return errors.Errorf("provided epoch participation json string is malformed: %s", string(b))
	}

	// Remove leading and trailing quotation marks.
	jsonString := string(b)
	jsonString = strings.Trim(jsonString, "\"")
	decoded, err := base64.StdEncoding.DecodeString(jsonString)
	if err != nil {
		return errors.Wrapf(err, "could not decode epoch participation base64 value")
	}

	*p = make([]string, len(decoded))
	for i, participation := range decoded {
		(*p)[i] = strconv.FormatUint(uint64(participation), 10)
	}
	return nil
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
