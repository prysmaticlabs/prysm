package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	types "github.com/prysmaticlabs/eth2-types"
	eth "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"google.golang.org/protobuf/types/known/emptypb"
)

func getStateCommandResult(command string, parameters []string) string {
	switch command {
	case genesisTime.command, genesisTime.shorthand:
		genesis, err := nodeClient.GetGenesis(context.Background(), &emptypb.Empty{})
		if err != nil {
			log.WithError(err).Error(err, "failed to get chain head")
			return ""
		}
		return fmt.Sprintf(genesisTime.responseText, time.Unix(genesis.GenesisTime.Seconds, 0))
	case beaconCommittee.command, beaconCommittee.shorthand:
		if len(parameters) != 2 {
			log.Error("Expected 2 parameters for committee command")
			return ""
		}
		req := &eth.ListCommitteesRequest{}
		committees, err := beaconClient.ListBeaconCommittees(context.Background(), req)
		if err != nil {
			log.WithError(err).Error(err, "failed to get committees")
			return ""
		}
		reqSlot, err := strconv.Atoi(parameters[0])
		if err != nil {
			log.WithError(err).Error(err, "failed to convert")
			return ""
		}
		reqIndex, err := strconv.Atoi(parameters[1])
		if err != nil {
			log.WithError(err).Error(err, "failed to convert")
			return ""
		}
		var resultCommittee []types.ValidatorIndex
		for slot, committeesForSlot := range committees.Committees {
			if slot == uint64(reqSlot) {
				resultCommittee = committeesForSlot.Committees[reqIndex].ValidatorIndices
			}
		}
		if len(resultCommittee) < 1 {
			return fmt.Sprintf("Committee of slot %d and index %d could not be calculated.", reqSlot, reqIndex)
		}
		return fmt.Sprintf(beaconCommittee.responseText, reqSlot, reqIndex, resultCommittee)
	default:
		return ""
	}
}
