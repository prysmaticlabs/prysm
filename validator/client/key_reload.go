package client

import (
	"context"

	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"go.opencensus.io/trace"
)

// HandleKeyReload makes sure the validator keeps operating correctly after a change to the underlying keys.
// It is also responsible for logging out information about the new state of keys.
func (v *validator) HandleKeyReload(ctx context.Context, newKeys [][48]byte) (anyActive bool, err error) {
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
	anyActive = v.checkAndLogValidatorStatus(statuses)
	if anyActive {
		logActiveValidatorStatus(statuses)
	}

	return anyActive, nil
}
