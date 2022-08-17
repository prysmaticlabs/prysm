package client

import (
	"context"

	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	eth "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"go.opencensus.io/trace"
)

// HandleKeyReload makes sure the validator keeps operating correctly after a change to the underlying keys.
// It is also responsible for logging out information about the new state of keys.
func (v *validator) HandleKeyReload(ctx context.Context, newKeys [][fieldparams.BLSPubkeyLength]byte) (anyActive bool, err error) {
	ctx, span := trace.StartSpan(ctx, "validator.HandleKeyReload")
	defer span.End()

	statusRequestKeys := make([][]byte, len(newKeys))
	for i := range newKeys {
		statusRequestKeys[i] = newKeys[i][:]
	}
	resp, err := v.validatorClient.MultipleValidatorStatus(ctx, &eth.MultipleValidatorStatusRequest{
		PublicKeys: statusRequestKeys,
	})
	if err != nil {
		return false, err
	}
	statuses := make([]*validatorStatus, len(resp.Statuses))
	for i, s := range resp.Statuses {
		statuses[i] = &validatorStatus{
			publicKey: resp.PublicKeys[i],
			status:    s,
			index:     resp.Indices[i],
		}
	}
	vals, err := v.beaconClient.ListValidators(ctx, &eth.ListValidatorsRequest{Active: true, PageSize: 0})
	if err != nil {
		return false, errors.Wrap(err, "could not get active validator count")
	}
	anyActive = v.checkAndLogValidatorStatus(statuses, uint64(vals.TotalSize))
	if anyActive {
		logActiveValidatorStatus(statuses)
	}

	return anyActive, nil
}
