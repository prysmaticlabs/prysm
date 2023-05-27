package client

import (
	"context"

	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"go.opencensus.io/trace"
)

// HandleKeyReload makes sure the validator keeps operating correctly after a change to the underlying keys.
// It is also responsible for logging out information about the new state of keys.
func (v *validator) HandleKeyReload(ctx context.Context, currentKeys [][fieldparams.BLSPubkeyLength]byte) (anyActive bool, err error) {
	ctx, span := trace.StartSpan(ctx, "validator.HandleKeyReload")
	defer span.End()

	statusRequestKeys := make([][]byte, len(currentKeys))
	for i := range currentKeys {
		statusRequestKeys[i] = currentKeys[i][:]
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

	anyActive = v.checkAndLogValidatorStatus(statuses)
	if anyActive {
		logActiveValidatorStatus(statuses)
	}

	return anyActive, nil
}
