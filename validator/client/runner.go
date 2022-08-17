package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	"github.com/prysmaticlabs/prysm/v3/validator/client/iface"
	"github.com/prysmaticlabs/prysm/v3/validator/keymanager"
	"github.com/prysmaticlabs/prysm/v3/validator/keymanager/remote"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// time to wait before trying to reconnect with beacon node.
var backOffPeriod = 10 * time.Second

// Run the main validator routine. This routine exits if the context is
// canceled.
//
// Order of operations:
// 1 - Initialize validator data
// 2 - Wait for validator activation
// 3 - Wait for the next slot start
// 4 - Update assignments
// 5 - Determine role at current slot
// 6 - Perform assigned role, if any
func run(ctx context.Context, v iface.Validator) {
	cleanup := v.Done
	defer cleanup()

	headSlot, err := waitForActivation(ctx, v)
	if err != nil {
		return // Exit if context is canceled.
	}

	connectionErrorChannel := make(chan error, 1)
	go v.ReceiveBlocks(ctx, connectionErrorChannel)
	if err := v.UpdateDuties(ctx, headSlot); err != nil {
		handleAssignmentError(err, headSlot)
	}

	accountsChangedChan := make(chan [][fieldparams.BLSPubkeyLength]byte, 1)
	km, err := v.Keymanager()
	if err != nil {
		log.WithError(err).Fatal("Could not get keymanager")
	}
	sub := km.SubscribeAccountChanges(accountsChangedChan)
	// Set properties on the beacon node like the fee recipient for validators that are being used & active.
	if err := v.PushProposerSettings(ctx, km); err != nil {
		if errors.Is(err, ErrBuilderValidatorRegistration) {
			log.WithError(err).Warn("Push proposer settings error")
		} else {
			log.WithError(err).Fatal("Failed to update proposer settings") // allow fatal. skipcq
		}
	}
	for {
		_, cancel := context.WithCancel(ctx)
		ctx, span := trace.StartSpan(ctx, "validator.processSlot")

		select {
		case <-ctx.Done():
			log.Info("Context canceled, stopping validator")
			span.End()
			cancel()
			sub.Unsubscribe()
			close(accountsChangedChan)
			return // Exit if context is canceled.
		case blocksError := <-connectionErrorChannel:
			if blocksError != nil {
				log.WithError(blocksError).Warn("block stream interrupted")
				go v.ReceiveBlocks(ctx, connectionErrorChannel)
				continue
			}
		case newKeys := <-accountsChangedChan:
			anyActive, err := v.HandleKeyReload(ctx, newKeys)
			if err != nil {
				log.WithError(err).Error("Could not properly handle reloaded keys")
			}
			if !anyActive {
				log.Info("No active keys found. Waiting for activation...")
				err := v.WaitForActivation(ctx, accountsChangedChan)
				if err != nil {
					log.WithError(err).Fatal("Could not wait for validator activation")
				}
			}
		case slot := <-v.NextSlot():
			span.AddAttributes(trace.Int64Attribute("slot", int64(slot))) // lint:ignore uintcast -- This conversion is OK for tracing.
			reloadRemoteKeys(ctx, km)
			allExited, err := v.AllValidatorsAreExited(ctx)
			if err != nil {
				log.WithError(err).Error("Could not check if validators are exited")
			}
			if allExited {
				log.Info("All validators are exited, no more work to perform...")
				continue
			}

			deadline := v.SlotDeadline(slot)
			slotCtx, cancel := context.WithDeadline(ctx, deadline)
			log := log.WithField("slot", slot)
			log.WithField("deadline", deadline).Debug("Set deadline for proposals and attestations")

			// Keep trying to update assignments if they are nil or if we are past an
			// epoch transition in the beacon node's state.
			if err := v.UpdateDuties(ctx, slot); err != nil {
				handleAssignmentError(err, slot)
				cancel()
				span.End()
				continue
			}

			if slots.IsEpochStart(slot) {
				go func() {
					//deadline set for next epoch rounded up
					if err := v.PushProposerSettings(ctx, km); err != nil {
						log.WithError(err).Warn("Failed to update proposer settings")
					}
				}()
			}

			// Start fetching domain data for the next epoch.
			if slots.IsEpochEnd(slot) {
				go v.UpdateDomainDataCaches(ctx, slot+1)
			}

			var wg sync.WaitGroup

			allRoles, err := v.RolesAt(ctx, slot)
			if err != nil {
				log.WithError(err).Error("Could not get validator roles")
				span.End()
				continue
			}
			performRoles(slotCtx, allRoles, v, slot, &wg, span)
		}
	}
}

func reloadRemoteKeys(ctx context.Context, km keymanager.IKeymanager) {
	remoteKm, ok := km.(remote.RemoteKeymanager)
	if ok {
		_, err := remoteKm.ReloadPublicKeys(ctx)
		if err != nil {
			log.WithError(err).Error(msgCouldNotFetchKeys)
		}
	}
}

func waitForActivation(ctx context.Context, v iface.Validator) (types.Slot, error) {
	ticker := time.NewTicker(backOffPeriod)
	defer ticker.Stop()

	var headSlot types.Slot
	firstTime := true
	for {
		if !firstTime {
			if ctx.Err() != nil {
				log.Info("Context canceled, stopping validator")
				return headSlot, errors.New("context canceled")
			}
			<-ticker.C
		} else {
			firstTime = false
		}
		err := v.WaitForChainStart(ctx)
		if isConnectionError(err) {
			log.WithError(err).Warn("Could not determine if beacon chain started")
			continue
		}
		if err != nil {
			log.WithError(err).Fatal("Could not determine if beacon chain started")
		}

		err = v.WaitForKeymanagerInitialization(ctx)
		if err != nil {
			// log.Fatal will prevent defer from being called
			v.Done()
			log.WithError(err).Fatal("Wallet is not ready")
		}

		err = v.WaitForSync(ctx)
		if isConnectionError(err) {
			log.WithError(err).Warn("Could not determine if beacon chain started")
			continue
		}
		if err != nil {
			log.WithError(err).Fatal("Could not determine if beacon node synced")
		}
		err = v.WaitForActivation(ctx, nil /* accountsChangedChan */)
		if isConnectionError(err) {
			log.WithError(err).Warn("Could not wait for validator activation")
			continue
		}
		if err != nil {
			log.WithError(err).Fatal("Could not wait for validator activation")
		}

		headSlot, err = v.CanonicalHeadSlot(ctx)
		if isConnectionError(err) {
			log.WithError(err).Warn("Could not get current canonical head slot")
			continue
		}
		if err != nil {
			log.WithError(err).Fatal("Could not get current canonical head slot")
		}
		err = v.CheckDoppelGanger(ctx)
		if isConnectionError(err) {
			log.WithError(err).Warn("Could not wait for checking doppelganger")
			continue
		}
		if err != nil {
			log.WithError(err).Fatal("Could not succeed with doppelganger check")
		}
		break
	}
	return headSlot, nil
}

func performRoles(slotCtx context.Context, allRoles map[[48]byte][]iface.ValidatorRole, v iface.Validator, slot types.Slot, wg *sync.WaitGroup, span *trace.Span) {
	for pubKey, roles := range allRoles {
		wg.Add(len(roles))
		for _, role := range roles {
			go func(role iface.ValidatorRole, pubKey [fieldparams.BLSPubkeyLength]byte) {
				defer wg.Done()
				switch role {
				case iface.RoleAttester:
					v.SubmitAttestation(slotCtx, slot, pubKey)
				case iface.RoleProposer:
					v.ProposeBlock(slotCtx, slot, pubKey)
				case iface.RoleAggregator:
					v.SubmitAggregateAndProof(slotCtx, slot, pubKey)
				case iface.RoleSyncCommittee:
					v.SubmitSyncCommitteeMessage(slotCtx, slot, pubKey)
				case iface.RoleSyncCommitteeAggregator:
					v.SubmitSignedContributionAndProof(slotCtx, slot, pubKey)
				case iface.RoleUnknown:
					log.WithField("pubKey", fmt.Sprintf("%#x", bytesutil.Trunc(pubKey[:]))).Trace("No active roles, doing nothing")
				default:
					log.Warnf("Unhandled role %v", role)
				}
			}(role, pubKey)
		}
	}

	// Wait for all processes to complete, then report span complete.
	go func() {
		wg.Wait()
		defer span.End()
		defer func() {
			if err := recover(); err != nil { // catch any panic in logging
				log.WithField("err", err).
					Error("Panic occurred when logging validator report. This" +
						" should never happen! Please file a report at github.com/prysmaticlabs/prysm/issues/new")
			}
		}()
		// Log this client performance in the previous epoch
		v.LogAttestationsSubmitted()
		v.LogSyncCommitteeMessagesSubmitted()
		if err := v.LogValidatorGainsAndLosses(slotCtx, slot); err != nil {
			log.WithError(err).Error("Could not report validator's rewards/penalties")
		}
	}()
}

func isConnectionError(err error) bool {
	return err != nil && errors.Is(err, iface.ErrConnectionIssue)
}

func handleAssignmentError(err error, slot types.Slot) {
	if errCode, ok := status.FromError(err); ok && errCode.Code() == codes.NotFound {
		log.WithField(
			"epoch", slot/params.BeaconConfig().SlotsPerEpoch,
		).Warn("Validator not yet assigned to epoch")
	} else {
		log.WithField("error", err).Error("Failed to update assignments")
	}
}
