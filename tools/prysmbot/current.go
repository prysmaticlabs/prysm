package main

import (
	"context"
	"fmt"

	eth "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/protobuf/types/known/emptypb"
)

func getHeadCommandResult(command string) string {
	switch command {
	case headSlot.command, headSlot.shorthand:
		chainHead, err := beaconClient.GetChainHead(context.Background(), &emptypb.Empty{})
		if err != nil {
			log.WithError(err).Error(err, "failed to get chain head")
			return "Could not get current slot."
		}
		return fmt.Sprintf(headSlot.responseText, chainHead.HeadSlot)
	case headEpoch.command, headEpoch.shorthand:
		chainHead, err := beaconClient.GetChainHead(context.Background(), &emptypb.Empty{})
		if err != nil {
			log.WithError(err).Error(err, "failed to get chain head")
			return "Could not get current epoch."
		}
		return fmt.Sprintf(headEpoch.responseText, chainHead.HeadEpoch)
	case headJustifiedEpoch.command, headJustifiedEpoch.shorthand:
		chainHead, err := beaconClient.GetChainHead(context.Background(), &emptypb.Empty{})
		if err != nil {
			log.WithError(err).Error(err, "failed to get chain head")
			return "Could not get current justified epoch."
		}
		return fmt.Sprintf(headJustifiedEpoch.responseText, chainHead.JustifiedEpoch)
	case headFinalizedEpoch.command, headFinalizedEpoch.shorthand:
		chainHead, err := beaconClient.GetChainHead(context.Background(), &emptypb.Empty{})
		if err != nil {
			log.WithError(err).Error(err, "failed to get chain head")
			return "Could not get current head finalized epoch."
		}
		return fmt.Sprintf(headFinalizedEpoch.responseText, chainHead.FinalizedEpoch)
	case currentParticipation.command, currentParticipation.shorthand, currentTotalBalance.command, currentTotalBalance.shorthand:
		req := &eth.GetValidatorParticipationRequest{}
		participation, err := beaconClient.GetValidatorParticipation(context.Background(), req)
		if err != nil {
			log.WithError(err).Error(err, "failed to get chain head")
			return "Could not get current participation/total balance."
		}
		if command == currentParticipation.command || command == currentParticipation.shorthand {
			return fmt.Sprintf(currentParticipation.responseText, participation.Epoch, participation.Participation.GlobalParticipationRate*100)
		} else if command == currentTotalBalance.command || command == currentTotalBalance.shorthand {
			inEther := float64(participation.Participation.EligibleEther) / float64(params.BeaconConfig().GweiPerEth)
			return fmt.Sprintf(currentTotalBalance.responseText, participation.Epoch, inEther)
		}
	default:
		return ""
	}
	return ""
}
