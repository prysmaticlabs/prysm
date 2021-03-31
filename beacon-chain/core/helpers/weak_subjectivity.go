package helpers

import (
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// WeakSubjectivityCheckptEpoch returns the epoch of the latest weak subjectivity checkpoint for the active validator count and
// finalized epoch.
//
// Reference spec implementation:
// https://github.com/ethereum/eth2.0-specs/blob/weak-subjectivity-guide/specs/phase0/weak-subjectivity.md#calculating-the-weak-subjectivity-period
//  def compute_weak_subjectivity_period(state):
//    weak_subjectivity_period = MIN_VALIDATOR_WITHDRAWABILITY_DELAY
//    val_count = len(get_active_validator_indices(state, get_current_epoch(state)))
//    if val_count >= MIN_PER_EPOCH_CHURN_LIMIT * CHURN_LIMIT_QUOTIENT:
//        weak_subjectivity_period += SAFETY_DECAY*CHURN_LIMIT_QUOTIENT/(2*100)
//    else:
//        weak_subjectivity_period += SAFETY_DECAY*val_count/(2*100*MIN_PER_EPOCH_CHURN_LIMIT)
//    return weak_subjectivity_period
func WeakSubjectivityCheckptEpoch(valCount uint64) (types.Epoch, error) {
	wsp := params.BeaconConfig().MinValidatorWithdrawabilityDelay

	m := params.BeaconConfig().MinPerEpochChurnLimit
	q := params.BeaconConfig().ChurnLimitQuotient
	d := params.BeaconConfig().SafetyDecay
	if valCount >= m*q {
		v := d * q / (2 * 100)
		wsp += types.Epoch(v)
	} else {
		v, err := mathutil.Mul64(d, valCount)
		if err != nil {
			return 0, err
		}
		v /= 2 * 100 * m
		wsp += types.Epoch(v)
	}
	return wsp, nil
}
