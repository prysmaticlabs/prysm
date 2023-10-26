package client

import (
	"context"
	"io"
	"time"

	validator2 "github.com/prysmaticlabs/prysm/v4/consensus-types/validator"
	"github.com/prysmaticlabs/prysm/v4/validator/client/iface"

	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v4/math"
	"github.com/prysmaticlabs/prysm/v4/monitoring/tracing"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	"go.opencensus.io/trace"
)

// WaitForActivation checks whether the validator pubkey is in the active
// validator set. If not, this operation will block until an activation message is
// received. This method also monitors the keymanager for updates while waiting for an activation
// from the gRPC server.
//
// If the channel parameter is nil, WaitForActivation creates and manages its own channel.
func (v *validator) WaitForActivation(ctx context.Context, accountsChangedChan chan [][fieldparams.BLSPubkeyLength]byte) error {
	// Monitor the key manager for updates.
	if accountsChangedChan == nil {
		accountsChangedChan = make(chan [][fieldparams.BLSPubkeyLength]byte, 1)
		km, err := v.Keymanager()
		if err != nil {
			return err
		}
		sub := km.SubscribeAccountChanges(accountsChangedChan)
		defer func() {
			sub.Unsubscribe()
			close(accountsChangedChan)
		}()
	}

	return v.internalWaitForActivation(ctx, accountsChangedChan)
}

// internalWaitForActivation performs the following:
// 1) While the key manager is empty, poll the key manager until some validator keys exist.
// 2) Open a server side stream for activation events against the given keys.
// 3) In another go routine, the key manager is monitored for updates and emits an update event on
// the accountsChangedChan. When an event signal is received, restart the internalWaitForActivation routine.
// 4) If the stream is reset in error, restart the routine.
// 5) If the stream returns a response indicating one or more validators are active, exit the routine.
func (v *validator) internalWaitForActivation(ctx context.Context, accountsChangedChan <-chan [][fieldparams.BLSPubkeyLength]byte) error {
	ctx, span := trace.StartSpan(ctx, "validator.WaitForActivation")
	defer span.End()

	validatingKeys, err := v.keyManager.FetchValidatingPublicKeys(ctx)
	if err != nil {
		return errors.Wrap(err, "could not fetch validating keys")
	}
	if len(validatingKeys) == 0 {
		log.Warn(msgNoKeysFetched)

		ticker := time.NewTicker(keyRefetchPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				validatingKeys, err = v.keyManager.FetchValidatingPublicKeys(ctx)
				if err != nil {
					return errors.Wrap(err, msgCouldNotFetchKeys)
				}
				if len(validatingKeys) == 0 {
					log.Warn(msgNoKeysFetched)
					continue
				}
			case <-ctx.Done():
				log.Debug("Context closed, exiting fetching validating keys")
				return ctx.Err()
			}
			break
		}
	}

	req := &ethpb.ValidatorActivationRequest{
		PublicKeys: bytesutil.FromBytes48Array(validatingKeys),
	}
	stream, err := v.validatorClient.WaitForActivation(ctx, req)
	if err != nil {
		tracing.AnnotateError(span, err)
		attempts := streamAttempts(ctx)
		log.WithError(err).WithField("attempts", attempts).
			Error("Stream broken while waiting for activation. Reconnecting...")
		// Reconnection attempt backoff, up to 60s.
		time.Sleep(time.Second * time.Duration(math.Min(uint64(attempts), 60)))
		return v.internalWaitForActivation(incrementRetries(ctx), accountsChangedChan)
	}

	if err = v.handleAccountsChanged(ctx, accountsChangedChan, &stream, span); err != nil {
		return err
	}

	v.ticker = slots.NewSlotTicker(time.Unix(int64(v.genesisTime), 0), params.BeaconConfig().SecondsPerSlot)
	return nil
}

func (v *validator) handleAccountsChanged(ctx context.Context, accountsChangedChan <-chan [][fieldparams.BLSPubkeyLength]byte, stream *ethpb.BeaconNodeValidator_WaitForActivationClient, span *trace.Span) error {
	for {
		select {
		case <-accountsChangedChan:
			// Accounts (keys) changed, restart the process.
			return v.internalWaitForActivation(ctx, accountsChangedChan)
		default:
			res, err := (*stream).Recv()
			// If the stream is closed, we stop the loop.
			if errors.Is(err, io.EOF) {
				break
			}
			// If context is canceled we return from the function.
			if ctx.Err() == context.Canceled {
				return errors.Wrap(ctx.Err(), "context has been canceled so shutting down the loop")
			}
			if err != nil {
				tracing.AnnotateError(span, err)
				attempts := streamAttempts(ctx)
				log.WithError(err).WithField("attempts", attempts).
					Error("Stream broken while waiting for activation. Reconnecting...")
				// Reconnection attempt backoff, up to 60s.
				time.Sleep(time.Second * time.Duration(math.Min(uint64(attempts), 60)))
				return v.internalWaitForActivation(incrementRetries(ctx), accountsChangedChan)
			}

			statuses := make([]*validatorStatus, len(res.Statuses))
			for i, s := range res.Statuses {
				statuses[i] = &validatorStatus{
					publicKey: s.PublicKey,
					status:    s.Status,
					index:     s.Index,
				}
			}

			// "-1" indicates that validator count endpoint is not supported by the beacon node.
			var valCount int64 = -1
			valCounts, err := v.prysmBeaconClient.GetValidatorCount(ctx, "head", []validator2.Status{validator2.Active})
			if err != nil && !errors.Is(err, iface.ErrNotSupported) {
				return errors.Wrap(err, "could not get active validator count")
			}

			if len(valCounts) > 0 {
				valCount = int64(valCounts[0].Count)
			}

			valActivated := v.checkAndLogValidatorStatus(statuses, valCount)
			if valActivated {
				logActiveValidatorStatus(statuses)
			} else {
				continue
			}
		}
		break
	}
	return nil
}

// Preferred way to use context keys is with a non built-in type. See: RVV-B0003
type waitForActivationContextKey string

const waitForActivationAttemptsContextKey = waitForActivationContextKey("WaitForActivation-attempts")

func streamAttempts(ctx context.Context) int {
	attempts, ok := ctx.Value(waitForActivationAttemptsContextKey).(int)
	if !ok {
		return 1
	}
	return attempts
}

func incrementRetries(ctx context.Context) context.Context {
	attempts := streamAttempts(ctx)
	return context.WithValue(ctx, waitForActivationAttemptsContextKey, attempts+1)
}
