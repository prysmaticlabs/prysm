package main

import (
	"context"
	"fmt"
	"strconv"

	eth "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func getValidatorCommandResult(command string, parameters []string) string {
	if len(parameters) != 1 {
		log.Error("Expected 1 parameter for validator command")
		return ""
	}
	reqIndex, err := strconv.Atoi(parameters[0])
	if err != nil {
		log.WithError(err).Error(err, "failed to convert")
		return ""
	}
	req := &eth.GetValidatorRequest{
		QueryFilter: &eth.GetValidatorRequest_Index{
			Index: uint64(reqIndex),
		},
	}
	switch command {
	case validatorBalance.command, validatorBalance.shorthand:
		balReq := &eth.ListValidatorBalancesRequest{
			Indices: []uint64{uint64(reqIndex)},
		}
		balances, err := beaconClient.ListValidatorBalances(context.Background(), balReq)
		if err != nil {
			log.WithError(err).Error(err, "failed to get balances")
			return ""
		}
		if len(balances.Balances) > 0 && balances.Balances[0].Index == uint64(reqIndex) {
			inEther := float64(balances.Balances[0].Balance) / float64(params.BeaconConfig().GweiPerEth)
			return fmt.Sprintf(validatorBalance.responseText, reqIndex, inEther)
		}
		return fmt.Sprintf("Could not get balance of valdiator index %d", reqIndex)
	case validatorActive.command, validatorActive.shorthand:
		validator, err := beaconClient.GetValidator(context.Background(), req)
		if err != nil {
			log.WithError(err).Error(err, "failed to get validator")
			return ""
		}
		return fmt.Sprintf(validatorActive.responseText, reqIndex, validator.ActivationEpoch)
	case validatorSlashed.command, validatorSlashed.shorthand:
		validator, err := beaconClient.GetValidator(context.Background(), req)
		if err != nil {
			log.WithError(err).Error(err, "failed to get validator")
			return ""
		}
		resultText := "not slashed"
		if validator.Slashed {
			resultText = "slashed"
		}
		return fmt.Sprintf(validatorSlashed.responseText, reqIndex, resultText)
	default:
		return ""
	}
}
