package beacon

import (
	"context"
	"io"
	"math/big"
	"sort"
	"sync"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// StreamValidatorsInfo returns a stream of information for given validators.
// Validators are supplied dynamically by the client, and can be added, removed and reset at any time.
func (bs *Server) StreamValidatorsInfo(stream ethpb.BeaconChain_StreamValidatorsInfoServer) error {
	pubKeys := make([][]byte, 0)
	pubKeysMutex := sync.RWMutex{}
	stateChannel := make(chan *feed.Event, 1)
	stateSub := bs.StateNotifier.StateFeed().Subscribe(stateChannel)
	depositBlocks := make(map[[48]byte]*big.Int)
	depositMutex := &sync.RWMutex{}
	defer stateSub.Unsubscribe()

	// Fetch our current epoch.
	headState, err := bs.HeadFetcher.HeadState(bs.Ctx)
	if err != nil {
		return status.Error(codes.Internal, "Could not access head state")
	}
	if headState == nil {
		return status.Error(codes.Internal, "Not ready to serve information")
	}
	currentEpoch := headState.Slot() / params.BeaconConfig().SlotsPerEpoch

	// Handle messages from client.
	go func() {
		for {
			msg, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				// Errors handle elsewhere
				select {
				case <-stream.Context().Done():
					return
				case <-bs.Ctx.Done():
					return
				case <-stateSub.Err():
					return
				default:
				}
				log.WithError(err).Debug("Receive from validators stream listener failed; client probably closed connection")
				return
			}
			switch msg.Action {
			case ethpb.SetAction_ADD_VALIDATOR_KEYS:
				pubKeysMutex.Lock()
				// Create existence map to ensure we don't duplicate keys.
				pubKeysMap := make(map[[48]byte]bool)
				for _, pubKey := range pubKeys {
					pubKeysMap[bytesutil.ToBytes48(pubKey)] = true
				}
				addedPubKeys := make([][]byte, 0)
				for _, pubKey := range msg.PublicKeys {
					if _, exists := pubKeysMap[bytesutil.ToBytes48(pubKey)]; !exists {
						pubKeys = append(pubKeys, pubKey)
						addedPubKeys = append(addedPubKeys, pubKey)
					}
				}
				pubKeysMutex.Unlock()
				// Send current info for the added public keys.
				if validators, err := bs.generateValidatorInfo(bs.Ctx, addedPubKeys, depositBlocks, depositMutex); err == nil {
					for _, validator := range validators {
						if err := stream.Send(validator); err != nil {
							stream.Context().Done()
							break
						}
					}
				}
			case ethpb.SetAction_REMOVE_VALIDATOR_KEYS:
				msgPubKeysMap := make(map[[48]byte]bool)
				for _, pubKey := range msg.PublicKeys {
					msgPubKeysMap[bytesutil.ToBytes48(pubKey)] = true
				}
				pubKeysMutex.Lock()
				max := len(pubKeys)
				for i := 0; i < max; i++ {
					if _, exists := msgPubKeysMap[bytesutil.ToBytes48(pubKeys[i])]; exists {
						copy(pubKeys[i:], pubKeys[i+1:])
						pubKeys = pubKeys[:len(pubKeys)-1]
						i--
						max--
					}
				}
				pubKeysMutex.Unlock()
			case ethpb.SetAction_SET_VALIDATOR_KEYS:
				pubKeysMutex.Lock()
				pubKeys = make([][]byte, 0, len(msg.PublicKeys))
				for _, pubKey := range msg.PublicKeys {
					pubKeys = append(pubKeys, pubKey)
				}
				pubKeysMutex.Unlock()
				if validators, err := bs.generateValidatorInfo(bs.Ctx, msg.PublicKeys, depositBlocks, depositMutex); err == nil {
					for _, validator := range validators {
						if err := stream.Send(validator); err != nil {
							stream.Context().Done()
							break
						}
					}
				}
			}
		}
	}()
	// Send responses at the end of every epoch.
	for {
		select {
		case event := <-stateChannel:
			if event.Type == statefeed.BlockProcessed {
				go func() {
					headState, err := bs.HeadFetcher.HeadState(bs.Ctx)
					if err != nil {
						log.Warn("Could not access head state for infostream")
						return
					}
					if headState == nil {
						// We aren't ready to serve information
						return
					}
					blockEpoch := headState.Slot() / params.BeaconConfig().SlotsPerEpoch
					if blockEpoch == currentEpoch {
						// Epoch hasn't changed, nothing to report yet.
						return
					}
					currentEpoch = blockEpoch
					pubKeysMutex.RLock()
					validators, err := bs.generateValidatorInfo(bs.Ctx, pubKeys, depositBlocks, depositMutex)
					pubKeysMutex.RUnlock()
					if err != nil {
						log.WithError(err).Warn("Failed to generate infostream response")
					}
					for _, validator := range validators {
						if err := stream.Send(validator); err != nil {
							// Client probably disconnected.
							log.WithError(err).Debug("Failed to send infostream response")
						}
					}
				}()
			}
		case <-stateSub.Err():
			return status.Error(codes.Aborted, "Subscriber closed")
		case <-bs.Ctx.Done():
			return status.Error(codes.Canceled, "Service context canceled")
		case <-stream.Context().Done():
			return status.Error(codes.Canceled, "Stream context canceled")
		}
	}
}

func (bs *Server) generateValidatorInfo(ctx context.Context, pubKeys [][]byte, depositBlocks map[[48]byte]*big.Int, depositMutex *sync.RWMutex) ([]*ethpb.ValidatorInfo, error) {
	headState, err := bs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not access head state")
	}
	if headState == nil {
		return nil, status.Error(codes.Internal, "Not ready to serve information")
	}
	epoch := headState.Slot() / params.BeaconConfig().SlotsPerEpoch
	if epoch == 0 {
		// Not reporting, but no error.
		return nil, nil
	}
	// We are reporting on the state at the end of the *previous* epoch.
	epoch--

	// pendingValidatorsMap is map from the validator pubkey to the index in our return array
	pendingValidatorsMap := make(map[[48]byte]int)
	genesisTime := headState.GenesisTime()
	validators := headState.ValidatorsReadOnly()
	res := make([]*ethpb.ValidatorInfo, 0)
	for _, pubKey := range pubKeys {
		info := &ethpb.ValidatorInfo{
			PublicKey: pubKey,
			Epoch:     epoch,
		}

		// Index
		var ok bool
		info.Index, ok, err = bs.BeaconDB.ValidatorIndex(ctx, pubKey)
		if err != nil {
			return nil, status.Error(codes.Internal, "Failed to obtain validator index")
		}
		if !ok {
			// We don't know of this validator; perhaps it's a pending deposit?
			var eth1BlockNumber *big.Int
			pubKey48 := bytesutil.ToBytes48(info.PublicKey)
			depositMutex.Lock()
			if cached, exists := depositBlocks[pubKey48]; exists {
				eth1BlockNumber = cached
			} else {
				_, fetched := bs.DepositFetcher.DepositByPubkey(bs.Ctx, info.PublicKey)
				depositBlocks[pubKey48] = fetched
				eth1BlockNumber = fetched
			}
			depositMutex.Unlock()
			if eth1BlockNumber != nil {
				info.Status = ethpb.ValidatorStatus_DEPOSITED
				if queueTimestamp, err := bs.depositQueueTimestamp(bs.Ctx, eth1BlockNumber, genesisTime); err != nil {
					log.WithError(err).Error("Failed to obtain queue activation timestamp")
				} else {
					info.TransitionTimestamp = queueTimestamp
				}
				res = append(res, info)
				continue
			}
			// We haven't found it; ignore this as it may be a deposit that hasn't been processed in Ethereum 1 yet.
		}
		validator := validators[info.Index]

		// Status and progression timestamp
		info.Status, info.TransitionTimestamp = bs.calculateStatusAndTransition(validator, headState, genesisTime)

		// TODO status timestamp
		// validator.StatusTimestamp

		// Balance
		info.Balance = headState.Balances()[info.Index]

		// Effective balance (for attesting states)
		if info.Status == ethpb.ValidatorStatus_ACTIVE ||
			info.Status == ethpb.ValidatorStatus_SLASHING ||
			info.Status == ethpb.ValidatorStatus_EXITING {
			info.EffectiveBalance = validator.EffectiveBalance()
		}

		// TODO Last attested slot
		// validator.LastAttestedSlot
		// TODO Next attesting slot
		// validator.NextAttestingSlot
		// TODO Last proposed slot
		// validator.LastProposedSlot
		// TODO Next proposing slot
		// validator.LastProposingSlot

		res = append(res, info)

		// Keep track of pending validators to fill in activation epoch later.
		if info.Status == ethpb.ValidatorStatus_PENDING {
			pendingValidatorsMap[bytesutil.ToBytes48(info.PublicKey)] = len(res) - 1
		}
	}

	// Calculate activation time for pending validators (if there are any).
	if len(pendingValidatorsMap) > 0 {
		numAttestingValidators := uint64(0)
		// Fetch the list of pending validators; count the number of attesting validators.
		pendingValidators := make([]uint64, 0, len(validators))
		for _, validator := range validators {
			if helpers.IsEligibleForActivationUsingTrie(headState, validator) {
				pubKey := validator.PublicKey()
				validatorIndex, ok, err := bs.BeaconDB.ValidatorIndex(ctx, pubKey[:])
				if err == nil && ok {
					pendingValidators = append(pendingValidators, validatorIndex)
				}
			}
			if helpers.IsActiveValidatorUsingTrie(validator, epoch) {
				numAttestingValidators++
			}
		}

		sortableIndices := &indicesSorter{
			validators: validators,
			indices:    pendingValidators,
		}
		sort.Sort(sortableIndices)

		sortedIndices := sortableIndices.indices

		// Loop over epochs, roughly simulating progression.
		for curEpoch := epoch + 1; len(sortedIndices) > 0 && len(pendingValidators) > 0; curEpoch++ {
			toProcess, _ := helpers.ValidatorChurnLimit(numAttestingValidators)
			if toProcess > uint64(len(sortedIndices)) {
				toProcess = uint64(len(sortedIndices))
			}
			for i := uint64(0); i < toProcess; i++ {
				validator := validators[sortedIndices[i]]
				if index, exists := pendingValidatorsMap[validator.PublicKey()]; exists {
					res[index].TransitionTimestamp = epochToTimestamp(genesisTime, helpers.DelayedActivationExitEpoch(curEpoch))
					delete(pendingValidatorsMap, validator.PublicKey())
				}
				numAttestingValidators++
			}
			sortedIndices = sortedIndices[toProcess:]
		}
	}

	return res, nil
}

type indicesSorter struct {
	validators []*state.ReadOnlyValidator
	indices    []uint64
}

func (s indicesSorter) Len() int      { return len(s.indices) }
func (s indicesSorter) Swap(i, j int) { s.indices[i], s.indices[j] = s.indices[j], s.indices[i] }
func (s indicesSorter) Less(i, j int) bool {
	if s.validators[s.indices[i]].ActivationEligibilityEpoch() == s.validators[s.indices[j]].ActivationEligibilityEpoch() {
		return s.indices[i] < s.indices[j]
	}
	return s.validators[s.indices[i]].ActivationEligibilityEpoch() < s.validators[s.indices[j]].ActivationEligibilityEpoch()
}

func (bs *Server) calculateStatusAndTransition(validator *state.ReadOnlyValidator, beaconState *state.BeaconState, genesisTime uint64) (ethpb.ValidatorStatus, uint64) {
	currentEpoch := helpers.CurrentEpoch(beaconState)
	farFutureEpoch := params.BeaconConfig().FarFutureEpoch

	if validator == nil {
		return ethpb.ValidatorStatus_UNKNOWN_STATUS, 0
	}

	if currentEpoch < validator.ActivationEligibilityEpoch() {
		if helpers.IsEligibleForActivationQueueUsingTrie(validator) {
			return ethpb.ValidatorStatus_DEPOSITED, epochToTimestamp(genesisTime, validator.ActivationEligibilityEpoch())
		}
		return ethpb.ValidatorStatus_DEPOSITED, 0
	}
	if currentEpoch < validator.ActivationEpoch() {
		return ethpb.ValidatorStatus_PENDING, epochToTimestamp(genesisTime, validator.ActivationEpoch())
	}
	if validator.ExitEpoch() == farFutureEpoch {
		return ethpb.ValidatorStatus_ACTIVE, 0
	}
	if currentEpoch < validator.ExitEpoch() {
		if validator.Slashed() {
			return ethpb.ValidatorStatus_SLASHING, epochToTimestamp(genesisTime, validator.ExitEpoch())
		}
		return ethpb.ValidatorStatus_EXITING, epochToTimestamp(genesisTime, validator.ExitEpoch())
	}
	return ethpb.ValidatorStatus_EXITED, epochToTimestamp(genesisTime, validator.WithdrawableEpoch())
}

// epochToTimestamp converts an epoch number to a timestamp.
func epochToTimestamp(genesisTime uint64, epoch uint64) uint64 {
	return genesisTime + epoch*params.BeaconConfig().SecondsPerSlot*params.BeaconConfig().SlotsPerEpoch
}

// depositQueueTimestamp calculates the timestamp for exit of the validator from the deposit queue.
func (bs *Server) depositQueueTimestamp(ctx context.Context, eth1BlockNumber *big.Int, genesisTime uint64) (uint64, error) {
	blockTimeStamp, err := bs.BlockFetcher.BlockTimeByHeight(ctx, eth1BlockNumber)
	if err != nil {
		return 0, err
	}
	followTime := time.Duration(params.BeaconConfig().Eth1FollowDistance*params.BeaconConfig().GoerliBlockTime) * time.Second
	eth1UnixTime := time.Unix(int64(blockTimeStamp), 0).Add(followTime)

	votingPeriod := time.Duration(params.BeaconConfig().SlotsPerEth1VotingPeriod*params.BeaconConfig().SecondsPerSlot) * time.Second
	activationTime := eth1UnixTime.Add(votingPeriod)
	eth2Genesis := time.Unix(int64(genesisTime), 0)

	if eth2Genesis.After(activationTime) {
		return genesisTime, nil
	}
	return uint64(activationTime.Unix()), nil
}
