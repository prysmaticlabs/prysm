package beacon

import (
	"encoding/hex"
	"fmt"
	types2 "github.com/gogo/protobuf/types"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var alreadySendEpochInfos map[types.Epoch]bool

// StreamMinimalConsensusInfo to orchestrator client every single time an unconfirmed block is received by the beacon node.
func (bs *Server) StreamMinimalConsensusInfo(
	req *ethpb.MinimalConsensusInfoRequest,
	stream ethpb.BeaconChain_StreamMinimalConsensusInfoServer,
) error {

	alreadySendEpochInfos = make(map[types.Epoch]bool)
	s, err := bs.HeadFetcher.HeadState(bs.Ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}

	currentEpoch := helpers.SlotToEpoch(s.Slot())
	requestedEpoch := req.FromEpoch
	if requestedEpoch > currentEpoch {
		log.WithField("curEpoch", currentEpoch).
			WithField("requestedEpoch", requestedEpoch).
			Warn("requested epoch is future from current epoch")
		return status.Errorf(codes.InvalidArgument, errEpoch, currentEpoch, requestedEpoch)
	}

	stateChannel := make(chan *feed.Event, 1)
	stateSub := bs.StateNotifier.StateFeed().Subscribe(stateChannel)
	defer stateSub.Unsubscribe()

	if err := bs.initialEpochInfoPropagation(requestedEpoch, currentEpoch, stream, stateChannel, stateSub); err != nil {
		log.WithError(err).Warn("Failed to send initial epoch infos to orchestrator")
		return err
	}

	for {
		select {
		case stateEvent := <-stateChannel:
			if stateEvent.Type == statefeed.BlockVerified {
				blockVerifiedData, ok := stateEvent.Data.(*statefeed.BlockPreVerifiedData)
				if !ok {
					log.Warn("Failed to send epoch info to orchestrator")
					continue
				}

				if err := bs.sendNextEpochInfo(blockVerifiedData.Slot, stream, blockVerifiedData.CurrentState); err != nil {
					log.WithField("epoch", helpers.SlotToEpoch(blockVerifiedData.Slot)+1).
						WithError(err).
						Warn("Failed to send epoch info to orchestrator")
					continue
				}
			}
		case <-stateSub.Err():
			return status.Error(codes.Aborted, "Subscriber closed, exiting go routine")
		case <-stream.Context().Done():
			return status.Error(codes.Canceled, "Stream context canceled")
		case <-bs.Ctx.Done():
			return status.Error(codes.Canceled, "RPC context canceled")
		}
	}
}

// initialEpochInfoPropagation
func (bs *Server) initialEpochInfoPropagation(
	requestedEpoch types.Epoch,
	currentEpoch types.Epoch,
	stream ethpb.BeaconChain_StreamMinimalConsensusInfoServer,
	stateChannel chan *feed.Event,
	stateSub event.Subscription,
) error {

	// when initial syncing is true, it starts sending epoch info
	if bs.SyncChecker.Syncing() {
		log.WithField("currentEpoch", currentEpoch).Debug("Node is in syncing mode")
		state, err := bs.HeadFetcher.HeadState(bs.Ctx)
		if err != nil {
			return err
		}

		epochInfo, err := bs.prepareEpochInfo(currentEpoch, state.Copy())
		if err != nil {
			log.WithField("epoch", currentEpoch).
				WithError(err).
				Warn("Failed to prepare epoch info in-sync mode")
			return status.Errorf(codes.Internal,
				"Could not prepare epoch info in-sync mode. epoch: %v  err: %v", currentEpoch, err)
		}

		if err := stream.Send(epochInfo); err != nil {
			return status.Errorf(codes.Unavailable,
				"Could not prepare epoch info in-sync mode. epoch: %v  err: %v", currentEpoch, err)
		}
		alreadySendEpochInfos[epochInfo.Epoch] = true

		for {
			select {
			case stateEvent := <-stateChannel:
				if stateEvent.Type == statefeed.BlockVerified {
					blockVerifiedData, ok := stateEvent.Data.(*statefeed.BlockPreVerifiedData)
					if !ok {
						continue
					}

					if err := bs.sendNextEpochInfo(blockVerifiedData.Slot, stream, blockVerifiedData.CurrentState); err != nil {
						log.WithField("epoch", helpers.SlotToEpoch(blockVerifiedData.Slot)+1).
							WithError(err).
							Warn("Failed to send initial epoch infos to orchestrator in-sync mode")
						continue
					}

					if !bs.SyncChecker.Syncing() {
						s, err := bs.HeadFetcher.HeadState(bs.Ctx)
						if err != nil {
							return status.Errorf(codes.Internal, "Could not get head state: %v", err)
						}
						currentEpoch := helpers.SlotToEpoch(s.Slot())
						nextEpoch := currentEpoch + 1

						log.WithField("epoch", currentEpoch).
							WithField("nextEpoch", nextEpoch).
							Info("Initial syncing done. sending next epoch info and exiting initial epochInfo propagation loop")

						epochInfo, err := bs.prepareEpochInfo(nextEpoch, s.Copy())
						if err != nil {
							log.WithField("epoch", nextEpoch).
								WithError(err).
								Warn("Failed to prepare epoch info")
							return err
						}

						if err := stream.Send(epochInfo); err != nil {
							return status.Errorf(codes.Unavailable,
								"Could not prepare epoch info in-sync mode. epoch: %v  err: %v", epochInfo.Epoch, err)
						}
						alreadySendEpochInfos[epochInfo.Epoch] = true
						return nil
					}
				}
			case <-stateSub.Err():
				return status.Error(codes.Aborted, "Subscriber closed, exiting initial epochInfo propagation loop")
			case <-bs.Ctx.Done():
				return status.Error(codes.Canceled, "Context canceled")
			case <-stream.Context().Done():
				return status.Error(codes.Canceled, "Context canceled")
			}
		}
	}

	log.WithField("currentEpoch", currentEpoch).Debug("Node is in non-syncing mode")
	state, err := bs.HeadFetcher.HeadState(bs.Ctx)
	if err != nil {
		return err
	}
	// sending past proposer assignments info to orchestrator
	for epoch := requestedEpoch; epoch <= currentEpoch; epoch++ {
		epochInfo, err := bs.prepareEpochInfo(epoch, state.Copy())
		if err != nil {
			log.WithField("epoch", epoch).
				WithError(err).
				Warn("Failed to prepare epoch info in non-syncing mode")
			return status.Errorf(codes.Internal,
				"Could not prepare epoch info in-sync mode. epoch: %v  err: %v", epoch, err)
		}

		if err := stream.Send(epochInfo); err != nil {
			return status.Errorf(codes.Unavailable,
				"Could not prepare epoch info non-sync mode. epoch: %v  err: %v", epoch, err)
		}

		alreadySendEpochInfos[epochInfo.Epoch] = true
	}

	return nil
}

// sendNextEpochInfo
func (bs *Server) sendNextEpochInfo(
	slot types.Slot,
	stream ethpb.BeaconChain_StreamMinimalConsensusInfoServer,
	s iface.BeaconState,
) error {
	epoch := helpers.SlotToEpoch(slot)
	nextEpoch := epoch + 1

	if !alreadySendEpochInfos[nextEpoch] {
		epochInfo, err := bs.prepareEpochInfo(nextEpoch, s)
		if err != nil {
			log.WithField("epoch", nextEpoch).
				WithError(err).
				Warn("Failed to prepare epoch info")
			return err
		}

		if err := stream.Send(epochInfo); err != nil {
			return status.Errorf(codes.Unavailable,
				"Could not prepare epoch info. epoch: %v  err: %v", epoch, err)
		}
		alreadySendEpochInfos[epochInfo.Epoch] = true
	}
	return nil
}

// prepareEpochInfo
func (bs *Server) prepareEpochInfo(epoch types.Epoch, s iface.BeaconState) (*ethpb.MinimalConsensusInfo, error) {
	// Advance state with empty transitions up to the requested epoch start slot.
	epochStartSlot, err := helpers.StartSlot(epoch)
	if err != nil {
		return nil, err
	}

	if s.Slot() < epochStartSlot {
		s, err = state.ProcessSlots(bs.Ctx, s, epochStartSlot)
		if err != nil {
			return nil, err
		}
	}

	startSlot, err := helpers.StartSlot(epoch)
	if err != nil {
		return nil, err
	}

	proposerAssignmentInfo, err := helpers.ProposerAssignments(s, epoch)
	if err != nil {
		return nil, err
	}

	epochStartTime, err := helpers.SlotToTime(uint64(bs.GenesisTimeFetcher.GenesisTime().Unix()), startSlot)
	if nil != err {
		return nil, err
	}

	validatorList, err := prepareSortedValidatorList(epoch, proposerAssignmentInfo)
	if err != nil {
		return nil, err
	}

	log.WithField("epoch", epoch).
		WithField("proposerList", fmt.Sprintf("%v", validatorList)).
		Debug("Successfully prepared proposer public key list")

	return &ethpb.MinimalConsensusInfo{
		Epoch:            epoch,
		ValidatorList:    validatorList,
		EpochTimeStart:   uint64(epochStartTime.Unix()),
		SlotTimeDuration: &types2.Duration{Seconds: int64(params.BeaconConfig().SecondsPerSlot)},
	}, nil
}

// prepareEpochInfo
func prepareSortedValidatorList(
	epoch types.Epoch,
	proposerAssignmentInfo []*ethpb.ValidatorAssignments_CommitteeAssignment,
) ([]string, error) {

	publicKeyList := make([]string, 0)
	slotToPubKeyMapping := make(map[types.Slot]string)

	for _, assignment := range proposerAssignmentInfo {
		for _, slot := range assignment.ProposerSlots {
			slotToPubKeyMapping[slot] = fmt.Sprintf("0x%s", hex.EncodeToString(assignment.PublicKey))
		}
	}

	if epoch == 0 {
		publicKeyBytes := make([]byte, params.BeaconConfig().BLSPubkeyLength)
		emptyPubKey := fmt.Sprintf("0x%s", hex.EncodeToString(publicKeyBytes))
		slotToPubKeyMapping[0] = emptyPubKey
	}

	startSlot, err := helpers.StartSlot(epoch)
	if err != nil {
		return []string{}, err
	}

	endSlot, err := helpers.EndSlot(epoch)
	if err != nil {
		return []string{}, err
	}

	for slot := startSlot; slot <= endSlot; slot++ {
		publicKeyList = append(publicKeyList, slotToPubKeyMapping[slot])
	}
	return publicKeyList, nil
}
