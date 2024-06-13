package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/api/client"
	"github.com/prysmaticlabs/prysm/v5/api/client/event"
	"github.com/prysmaticlabs/prysm/v5/config/features"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
	"github.com/prysmaticlabs/prysm/v5/validator/client/iface"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Time to wait before trying to reconnect with beacon node.
var backOffPeriod = 10 * time.Second

// run is the main validator routine. It runs indefinitely until the context is canceled.
//
// Main responsibilities of the routine:
// a) fetch and perform validator roles every slot,
// b) periodically check if the beacon node is healthy and perform a switchover if possible
// c) push proposer settings when appropriate
// d) process events
// e) act on account changes
func run(ctx context.Context, v iface.Validator) {
	cleanup := v.Done
	defer cleanup()

	if err := initializeValidator(ctx, v); err != nil {
		log.WithError(err).Error("Could not initialize validator")
		return
	}
	headSlot, err := v.CanonicalHeadSlot(ctx)
	if err != nil {
		log.WithError(err).Error("Could not get canonical head slot")
		return
	}

	km, err := v.Keymanager()
	if err != nil {
		log.WithError(err).Error("Could not get keymanager")
		return
	}
	accountsChangedChan := make(chan [][fieldparams.BLSPubkeyLength]byte, 1)
	sub := km.SubscribeAccountChanges(accountsChangedChan)

	if v.ProposerSettings() == nil {
		log.Warn("Validator client started without proposer settings such as fee recipient" +
			" and will continue to use settings provided in the beacon node.")
	}

	pushProposerSettingsChan := make(chan primitives.Slot, 1)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case slot := <-pushProposerSettingsChan:
				go func() {
					psCtx, psCancel := context.WithDeadline(ctx, time.Now().Add(5*time.Minute))
					if err := v.PushProposerSettings(psCtx, km, slot); err != nil {
						log.WithError(err).Warn("Failed to update proposer settings")
					}
					// release resources
					psCancel()
				}()
			}
		}
	}()
	pushProposerSettingsChan <- headSlot

	updateDutiesNeeded := false
	if err = v.UpdateDuties(ctx, headSlot); err != nil {
		handleUpdateDutiesError(err, headSlot)
		updateDutiesNeeded = true
	}

	eventsChan := make(chan *event.Event, 1)
	go v.StartEventStream(ctx, event.DefaultEventTopics, eventsChan)

	var (
		slotSpan             *trace.Span
		slotCtx              context.Context
		slotCancel           context.CancelFunc
		initializationNeeded = false
		nodeIsHealthyPrev    = true
	)

	for {
		select {
		case <-ctx.Done():
			log.Info("Context canceled, stopping validator")
			sub.Unsubscribe()
			close(accountsChangedChan)
			close(eventsChan)
			if slotCancel != nil {
				slotCancel()
			}
			return
		case slot := <-v.NextSlot(): // This case should be always handled first so that other actions don't block slot processing
			if !nodeIsHealthyPrev {
				continue
			}
			if initializationNeeded {
				initializationNeeded = false
				if err := initializeValidator(ctx, v); err != nil {
					log.WithError(err).Error("Could not initialize validator")
					if slotCancel != nil {
						slotCancel()
					}
					return
				}
			}

			slotCtx, slotCancel = context.WithDeadline(ctx, v.SlotDeadline(slot))
			slotCtx, slotSpan = trace.StartSpan(slotCtx, "validator.processSlot")
			slotSpan.AddAttributes(trace.Int64Attribute("slot", int64(slot))) // lint:ignore uintcast -- This conversion is OK for tracing.

			if slots.IsEpochStart(slot) {
				updateDutiesNeeded = true
				// Update proposer settings at the start of each epoch because new validators could have become active
				pushProposerSettingsChan <- slot
			}

			if updateDutiesNeeded {
				updateDutiesNeeded = false
				if err = v.UpdateDuties(ctx, slot); err != nil {
					handleUpdateDutiesError(err, slot)
					updateDutiesNeeded = true
					slotCancel()
					slotSpan.End()
					continue
				}
			}

			var wg sync.WaitGroup
			roles, err := v.RolesAt(slotCtx, slot)
			if err != nil {
				log.WithError(err).Error("Could not get validator roles")
				slotCancel()
				slotSpan.End()
				continue
			}
			performRoles(slotCtx, roles, v, slot, &wg, slotSpan)

			// Start fetching domain data for the next epoch.
			if slots.IsEpochEnd(slot) {
				go v.UpdateDomainDataCaches(ctx, slot+1)
			}
		case slot := <-v.LastSecondOfSlot():
			nodeIsHealthyCurr := v.HealthTracker().IsHealthy()
			if nodeIsHealthyCurr {
				if !nodeIsHealthyPrev {
					pushProposerSettingsChan <- slot
				}
				// In case event stream died
				if !v.EventStreamIsRunning() {
					log.Info("Event stream reconnecting...")
					go v.StartEventStream(ctx, event.DefaultEventTopics, eventsChan)
				}
			} else if features.Get().EnableBeaconRESTApi {
				v.ChangeHost()
				initializationNeeded = true
				updateDutiesNeeded = true
				if slotCancel != nil {
					slotCancel()
				}
			}

			nodeIsHealthyPrev = nodeIsHealthyCurr
		case e := <-eventsChan:
			v.ProcessEvent(e)
		case keys := <-accountsChangedChan:
			onAccountsChanged(ctx, v, keys, accountsChangedChan)
			updateDutiesNeeded = true
			headSlot, err := v.CanonicalHeadSlot(ctx)
			if err != nil {
				log.WithError(err).Error("Could not get canonical head slot")
				continue
			}
			pushProposerSettingsChan <- headSlot
		}
	}
}

func onAccountsChanged(ctx context.Context, v iface.Validator, current [][48]byte, ac chan [][fieldparams.BLSPubkeyLength]byte) {
	anyActive, err := v.HandleKeyReload(ctx, current)
	if err != nil {
		log.WithError(err).Error("Could not properly handle reloaded keys")
		return
	}
	if !anyActive {
		log.Warn("No active keys found. Waiting for activation...")
		err = v.WaitForActivation(ctx, ac)
		if err != nil {
			log.WithError(err).Warn("Could not wait for validator activation")
		}
	}
}

func initializeValidator(ctx context.Context, v iface.Validator) error {
	ticker := time.NewTicker(backOffPeriod)
	defer ticker.Stop()

	firstTime := true

	for {
		if !firstTime {
			if ctx.Err() != nil {
				log.Info("Context canceled, stopping validator")
				return errors.New("context canceled")
			}
			<-ticker.C
		}

		firstTime = false

		if err := v.WaitForChainStart(ctx); err != nil {
			if isConnectionError(err) {
				log.WithError(err).Warn("Could not determine if beacon chain started")
				continue
			}

			log.WithError(err).Fatal("Could not determine if beacon chain started")
		}

		if err := v.WaitForKeymanagerInitialization(ctx); err != nil {
			// log.Fatal will prevent defer from being called
			v.Done()
			log.WithError(err).Fatal("Wallet is not ready")
		}

		if err := v.WaitForSync(ctx); err != nil {
			if isConnectionError(err) {
				log.WithError(err).Warn("Could not determine if beacon chain started")
				continue
			}

			log.WithError(err).Fatal("Could not determine if beacon node synced")
		}

		if err := v.WaitForActivation(ctx, nil /* accountsChangedChan */); err != nil {
			log.WithError(err).Fatal("Could not wait for validator activation")
		}

		if err := v.CheckDoppelGanger(ctx); err != nil {
			if isConnectionError(err) {
				log.WithError(err).Warn("Could not wait for checking doppelganger")
				continue
			}

			log.WithError(err).Fatal("Could not succeed with doppelganger check")
		}
		break
	}
	return nil
}

func performRoles(slotCtx context.Context, allRoles map[[48]byte][]iface.ValidatorRole, v iface.Validator, slot primitives.Slot, wg *sync.WaitGroup, span *trace.Span) {
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
					log.WithField("pubkey", fmt.Sprintf("%#x", bytesutil.Trunc(pubKey[:]))).Trace("No active roles, doing nothing")
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
				log.WithField("error", err).
					Error("Panic occurred when logging validator report. This" +
						" should never happen! Please file a report at github.com/prysmaticlabs/prysm/issues/new")
			}
		}()
		// Log performance in the previous slot
		v.LogSubmittedAtts(slot)
		v.LogSubmittedSyncCommitteeMessages()
		if err := v.LogValidatorGainsAndLosses(slotCtx, slot); err != nil {
			log.WithError(err).Error("Could not report validator's rewards/penalties")
		}
	}()
}

func isConnectionError(err error) bool {
	return err != nil && errors.Is(err, client.ErrConnectionIssue)
}

func handleUpdateDutiesError(err error, slot primitives.Slot) {
	if errors.Is(err, ErrValidatorsAllExited) {
		log.Warn(ErrValidatorsAllExited)
	} else if errCode, ok := status.FromError(err); ok && errCode.Code() == codes.NotFound {
		log.WithField(
			"epoch", slot/params.BeaconConfig().SlotsPerEpoch,
		).Warn("Validator not yet assigned to epoch")
	} else {
		log.WithError(err).Error("Failed to update duties")
	}
}
