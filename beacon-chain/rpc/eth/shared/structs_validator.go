package shared

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

type ValidatorPerformanceResponse struct {
	CurrentEffectiveBalances      []uint64 `json:"current_effective_balances"`
	InclusionSlots                []uint64 `json:"inclusion_slots"`
	InclusionDistances            []uint64 `json:"inclusion_distances"`
	CorrectlyVotedSource          []bool   `json:"correctly_voted_source"`
	CorrectlyVotedTarget          []bool   `json:"correctly_voted_target"`
	CorrectlyVotedHead            []bool   `json:"correctly_voted_head"`
	BalancesBeforeEpochTransition []uint64 `json:"balances_before_epoch_transition"`
	BalancesAfterEpochTransition  []uint64 `json:"balances_after_epoch_transition"`
	MissingValidators             []string `json:"missing_validators"`
	AverageActiveValidatorBalance float32  `json:"average_active_validator_balance"`
	PublicKeys                    []string `json:"public_keys"`
	InactivityScores              []uint64 `json:"inactivity_scores"`
}

func ValidatorPerformanceResponseFromConsensus(e *eth.ValidatorPerformanceResponse) *ValidatorPerformanceResponse {
	inclusionSlots := make([]uint64, len(e.InclusionSlots))
	for i, index := range e.InclusionSlots {
		inclusionSlots[i] = uint64(index)
	}
	inclusionDistances := make([]uint64, len(e.InclusionDistances))
	for i, index := range e.InclusionDistances {
		inclusionDistances[i] = uint64(index)
	}
	missingValidators := make([]string, len(e.MissingValidators))
	for i, key := range e.MissingValidators {
		missingValidators[i] = hexutil.Encode(key)
	}
	publicKeys := make([]string, len(e.PublicKeys))
	for i, key := range e.PublicKeys {
		publicKeys[i] = hexutil.Encode(key)
	}
	if len(e.CurrentEffectiveBalances) == 0 {
		e.CurrentEffectiveBalances = make([]uint64, 0)
	}
	if len(e.BalancesBeforeEpochTransition) == 0 {
		e.BalancesBeforeEpochTransition = make([]uint64, 0)
	}
	if len(e.BalancesAfterEpochTransition) == 0 {
		e.BalancesAfterEpochTransition = make([]uint64, 0)
	}
	if len(e.CorrectlyVotedSource) == 0 {
		e.CorrectlyVotedSource = make([]bool, 0)
	}
	if len(e.CorrectlyVotedTarget) == 0 {
		e.CorrectlyVotedTarget = make([]bool, 0)
	}
	if len(e.CorrectlyVotedHead) == 0 {
		e.CorrectlyVotedHead = make([]bool, 0)
	}
	if len(e.InactivityScores) == 0 {
		e.InactivityScores = make([]uint64, 0)
	}
	return &ValidatorPerformanceResponse{
		CurrentEffectiveBalances:      e.CurrentEffectiveBalances,
		InclusionSlots:                inclusionSlots,
		InclusionDistances:            inclusionDistances,
		CorrectlyVotedSource:          e.CorrectlyVotedSource,
		CorrectlyVotedTarget:          e.CorrectlyVotedTarget,
		CorrectlyVotedHead:            e.CorrectlyVotedHead,
		BalancesBeforeEpochTransition: e.BalancesBeforeEpochTransition,
		BalancesAfterEpochTransition:  e.BalancesAfterEpochTransition,
		MissingValidators:             missingValidators,
		AverageActiveValidatorBalance: e.AverageActiveValidatorBalance,
		PublicKeys:                    publicKeys,
		InactivityScores:              e.InactivityScores,
	}
}

type ValidatorBalancesResponse struct {
	Epoch         uint64              `json:"epoch"`
	Balances      []*ValidatorBalance `json:"balances"`
	NextPageToken string              `json:"next_page_token"`
	TotalSize     int32               `json:"total_size,omitempty"`
}

type ValidatorBalance struct {
	PublicKey string `json:"public_key"`
	Index     uint64 `json:"index"`
	Balance   uint64 `json:"balance"`
	Status    string `json:"status"`
}

func ValidatorBalancesResponseFromConsensus(e *eth.ValidatorBalances) (*ValidatorBalancesResponse, error) {
	balances := make([]*ValidatorBalance, len(e.Balances))
	for i, balance := range e.Balances {
		balances[i] = &ValidatorBalance{
			PublicKey: hexutil.Encode(balance.PublicKey),
			Index:     uint64(balance.Index),
			Balance:   balance.Balance,
			Status:    balance.Status,
		}
	}
	return &ValidatorBalancesResponse{
		Epoch:         uint64(e.Epoch),
		Balances:      balances,
		NextPageToken: e.NextPageToken,
		TotalSize:     e.TotalSize,
	}, nil
}

type ValidatorsResponse struct {
	Epoch         uint64                `json:"epoch"`
	ValidatorList []*ValidatorContainer `json:"validator_list"`
	NextPageToken string                `json:"next_page_token"`
	TotalSize     int32                 `json:"total_size"`
}

type ValidatorContainer struct {
	Index     uint64     `json:"index"`
	Validator *Validator `json:"validator"`
}

type Validator struct {
	PublicKey                  string `json:"public_key,omitempty"`
	WithdrawalCredentials      string `json:"withdrawal_credentials"`
	EffectiveBalance           uint64 `json:"effective_balance"`
	Slashed                    bool   `json:"slashed"`
	ActivationEligibilityEpoch uint64 `json:"activation_eligibility_epoch"`
	ActivationEpoch            uint64 `json:"activation_epoch"`
	ExitEpoch                  uint64 `json:"exit_epoch"`
	WithdrawableEpoch          uint64 `json:"withdrawable_epoch"`
}

func ValidatorsResponseFromConsensus(e *eth.Validators) (*ValidatorsResponse, error) {
	validatorList := make([]*ValidatorContainer, len(e.ValidatorList))
	for i, validatorContainer := range e.ValidatorList {
		val := validatorContainer.Validator
		validatorList[i] = &ValidatorContainer{
			Index: uint64(validatorContainer.Index),
			Validator: &Validator{
				PublicKey:                  hexutil.Encode(val.PublicKey),
				WithdrawalCredentials:      hexutil.Encode(val.WithdrawalCredentials),
				EffectiveBalance:           val.EffectiveBalance,
				Slashed:                    val.Slashed,
				ActivationEligibilityEpoch: uint64(val.ActivationEligibilityEpoch),
				ActivationEpoch:            uint64(val.ActivationEpoch),
				ExitEpoch:                  uint64(val.ExitEpoch),
				WithdrawableEpoch:          uint64(val.WithdrawableEpoch),
			},
		}
	}
	return &ValidatorsResponse{
		Epoch:         uint64(e.Epoch),
		ValidatorList: validatorList,
		NextPageToken: e.NextPageToken,
		TotalSize:     e.TotalSize,
	}, nil
}
