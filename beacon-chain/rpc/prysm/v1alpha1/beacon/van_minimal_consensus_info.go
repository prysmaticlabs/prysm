package beacon

import (
	"encoding/hex"
	"fmt"
	duration "github.com/golang/protobuf/ptypes/duration"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// latestSendEpoch tracks last sent epoch number
var latestSendEpoch types.Epoch

// StreamMinimalConsensusInfo to orchestrator client every single time an unconfirmed block is received by the beacon node.
func (bs *Server) StreamMinimalConsensusInfo(
	req *ethpb.MinimalConsensusInfoRequest,
	stream ethpb.BeaconChain_StreamMinimalConsensusInfoServer,
) error {

	sender := func(epoch types.Epoch, state iface.BeaconState) error {
		if latestSendEpoch == 0 || latestSendEpoch != epoch {
			epochInfo, err := bs.prepareEpochInfo(epoch, state)
			if err != nil {
				return status.Errorf(codes.Internal,
					"Could not send over stream: %v", err)
			}
			if err := stream.Send(epochInfo); err != nil {
				return status.Errorf(codes.Unavailable,
					"Could not send over stream: %v  err: %v", epoch, err)
			}
			log.WithField("epoch", epoch).Info("Published epoch info")
			log.WithField("epoch", epoch).
				WithField("epochInfo", fmt.Sprintf("%+v", epochInfo)).
				Debug("Sent epoch info")
			latestSendEpoch = epoch
		}
		return nil
	}

	batchSender := func(start, end types.Epoch) error {
		for i := start; i <= end; i++ {
			startSlot, err := helpers.StartSlot(i)
			if err != nil {
				return status.Errorf(codes.Internal,
					"Could not send over stream: %v", err)
			}
			state, err := bs.StateGen.StateBySlot(bs.Ctx, startSlot)
			if err != nil {
				return status.Errorf(codes.Internal,
					"Could not send over stream: %v", err)
			}
			if !state.IsNil() {
				if err := sender(i, state.Copy()); err != nil {
					return err
				}
			}
		}
		return nil
	}

	cp, err := bs.BeaconDB.FinalizedCheckpoint(bs.Ctx)
	if err != nil {
		return status.Errorf(codes.Internal,
			"Could not send over stream: %v", err)
	}

	startEpoch := req.FromEpoch
	endEpoch := cp.Epoch
	latestSendEpoch = req.FromEpoch
	if latestSendEpoch > 0 {
		latestSendEpoch--
	}

	log.WithField("startEpoch", startEpoch).
		WithField("endEpoch", endEpoch).
		Info("Publishing previous epoch infos")

	if startEpoch == 0 || startEpoch < endEpoch {
		if err := batchSender(startEpoch, endEpoch); err != nil {
			return err
		}
	}

	stateChannel := make(chan *feed.Event, 1)
	stateSub := bs.StateNotifier.StateFeed().Subscribe(stateChannel)
	firstTime := true
	defer stateSub.Unsubscribe()

	for {
		select {
		case stateEvent := <-stateChannel:
			if stateEvent.Type == statefeed.BlockVerified {
				blockVerifiedData, ok := stateEvent.Data.(*statefeed.BlockPreVerifiedData)
				if !ok {
					log.Warn("Failed to send epoch info to orchestrator")
					continue
				}
				curEpoch := helpers.SlotToEpoch(blockVerifiedData.Slot)
				nextEpoch := curEpoch + 1
				// Executes for a single time
				if firstTime {
					firstTime = false
					log.WithField("startEpoch", latestSendEpoch+1).
						WithField("endEpoch", curEpoch).
						WithField("liveSyncStart", curEpoch+1).
						Info("Publishing left over epoch infos")

					startEpoch = latestSendEpoch + 1
					endEpoch = curEpoch
					curState := blockVerifiedData.CurrentState

					for i := startEpoch; i <= endEpoch; i++ {
						if err := sender(i, curState); err != nil {
							return err
						}
					}
				}
				if err := sender(nextEpoch, blockVerifiedData.CurrentState); err != nil {
					return err
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

// prepareEpochInfo
func (bs *Server) prepareEpochInfo(epoch types.Epoch, s iface.BeaconState) (*ethpb.MinimalConsensusInfo, error) {
	// Advance state with empty transitions up to the requested epoch start slot.
	startSlot, err := helpers.StartSlot(epoch)
	if err != nil {
		return nil, err
	}
	if s.Slot() < startSlot {
		s, err = state.ProcessSlots(bs.Ctx, s, startSlot)
		if err != nil {
			return nil, err
		}
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

	return &ethpb.MinimalConsensusInfo{
		Epoch:            epoch,
		ValidatorList:    validatorList,
		EpochTimeStart:   uint64(epochStartTime.Unix()),
		SlotTimeDuration: &duration.Duration{Seconds: int64(params.BeaconConfig().SecondsPerSlot)},
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
