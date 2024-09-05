package client

import (
	"context"

	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	validator2 "github.com/prysmaticlabs/prysm/v5/consensus-types/validator"
	"github.com/prysmaticlabs/prysm/v5/monitoring/tracing/trace"
	"github.com/prysmaticlabs/prysm/v5/validator/client/iface"
)

// HandleKeyReload makes sure the validator keeps operating correctly after a change to the underlying keys.
// It is also responsible for logging out information about the new state of keys.
func (v *validator) HandleKeyReload(ctx context.Context, currentKeys [][fieldparams.BLSPubkeyLength]byte) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "validator.HandleKeyReload")
	defer span.End()

	if err := v.updateValidatorStatusCache(ctx, currentKeys); err != nil {
		return false, err
	}

	// "-1" indicates that validator count endpoint is not supported by the beacon node.
	var valCount int64 = -1
	valCounts, err := v.prysmChainClient.ValidatorCount(ctx, "head", []validator2.Status{validator2.Active})
	if err != nil && !errors.Is(err, iface.ErrNotSupported) {
		return false, errors.Wrap(err, "could not get active validator count")
	}

	if len(valCounts) > 0 {
		valCount = int64(valCounts[0].Count)
	}

	return v.checkAndLogValidatorStatus(valCount), nil
}
