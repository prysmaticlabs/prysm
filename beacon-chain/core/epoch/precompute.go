package epoch

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// ValidatorInfo track the pre-compute of individual validator info such
// whether validator has performed certain action. (i.e. votes, block inclusion,
// winning root participation)
type ValidatorInfo struct {
	// IsSlashed is true if the validator has been slashed.
	IsSlashed bool
	// IsWithdrawableCurrentEpoch is true if the validator can withdraw current epoch.
	IsWithdrawableCurrentEpoch bool
	// IsActiveCurrentEpoch is true if the validator was active current epoch.
	IsActiveCurrentEpoch bool
	// IsActivePrevEpoch is true if the validator was active prev epoch.
	IsActivePrevEpoch bool
	// IsCurrentEpochAttester is true if the validator attested current epoch.
	IsCurrentEpochAttester bool
	// IsCurrentEpochTargetAttester is true if the validator attested current epoch target.
	IsCurrentEpochTargetAttester bool
	// IsPrevEpochAttester is true if the validator attested previous epoch.
	IsPrevEpochAttester bool
	// IsPrevEpochTargetAttester is true if the validator attested previous epoch target.
	IsPrevEpochTargetAttester bool
	// IsHeadAttester is true if the validator attested head.
	IsHeadAttester bool
	// IsProposer is true if the validator was a proposer.
	IsProposer bool
	// CurrentEpochEffectiveBalance is how much effective balance a validator has current epoch.
	CurrentEpochEffectiveBalance uint64

	// InclusionSlot is the earliest slot a validator had an attestation included in prev epoch.
	InclusionSlot uint64
	// InclusionDistance is the distance between the attestation slot and attestation was included in block.
	InclusionDistance uint64
	// ProposerIndex is the index of proposer at slot where attestation was included.
	ProposerIndex uint64

	//  WinningRootCommitteeBalance is the total balance of the crosslink committee.
	WinningRootCommitteeBalance uint64
	// WinningRootAttestingBalance is the total balance of the crosslink committee tht attested for winning root.
	WinningRootAttestingBalance uint64
}

// BalanceInfo track the different sets of balances during prev and current epochs.
type BalanceInfo struct {
	// CurrentEpoch is the total effective balance of all active validators during current epoch.
	CurrentEpoch uint64
	// PrevEpoch is the total effective balance of all active validators during prev epoch.
	PrevEpoch uint64
	// CurrentEpochAttesters is the total effective balance of all validators who attested during current epoch.
	CurrentEpochAttesters uint64
	// CurrentEpochTargetAttesters is the total effective balance of all validators who attested
	// for epoch boundary block during current epoch.
	CurrentEpochTargetAttesters uint64
	// PrevEpochAttesters is the total effective balance of all validators who attested during prev epoch.
	PrevEpochAttesters uint64
	// PrevEpochTargetAttesters is the total effective balance of all validators who attested
	// for epoch boundary block during prev epoch.
	PrevEpochTargetAttesters uint64
	// PrevEpochHeadAttesters is the total effective balance of all validators who attested
	// correctly for head block during prev epoch.
	PrevEpochHeadAttesters uint64
}

// NewPrecompute returns new instances of ValidatorInfo and BalanceInfo.
func NewPrecompute(state *pb.BeaconState) ([]*ValidatorInfo, *BalanceInfo) {
	vInfo := make([]*ValidatorInfo, len(state.Validators))
	bInfo := &BalanceInfo{}

	for i, v := range state.Validators {
		currentEpoch := helpers.CurrentEpoch(state)
		prevEpoch := helpers.PrevEpoch(state)
		withdrawable := currentEpoch > v.WithdrawableEpoch
		info := &ValidatorInfo{
			IsSlashed:                    v.Slashed,
			IsWithdrawableCurrentEpoch:   withdrawable,
			CurrentEpochEffectiveBalance: v.EffectiveBalance,
		}
		if helpers.IsActiveValidator(v, currentEpoch) {
			info.IsActiveCurrentEpoch = true
			bInfo.CurrentEpoch += v.EffectiveBalance
		}
		if helpers.IsActiveValidator(v, prevEpoch) {
			info.IsActivePrevEpoch = true
			bInfo.PrevEpoch += v.EffectiveBalance
		}
		vInfo[i] = info
	}
	return vInfo, bInfo
}
