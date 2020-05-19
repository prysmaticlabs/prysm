package accounts

import (
	"context"
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// ValidatorStatusMetadata holds all status information about a validator.
type ValidatorStatusMetadata struct {
	PublicKey []byte
	Metadata  *ethpb.ValidatorStatusResponse
}

// RunStatusCommand is the entry point to the `validator status` command.
func RunStatusCommand(
	pubkeys [][]byte, beaconNodeRPCProvider ethpb.BeaconNodeValidatorClient) error {
	statuses, err := FetchAccountStatuses(
		context.Background(), beaconNodeRPCProvider, pubkeys)
	if err != nil {
		return errors.Wrap(err, "Could not fetch account statuses from the beacon node")
	}
	printStatuses(statuses)
	return nil
}

// FetchAccountStatuses fetches validator statuses from the BeaconNodeValidatorClient
// for each validator public key.
func FetchAccountStatuses(
	ctx context.Context,
	beaconNodeRPCProvider ethpb.BeaconNodeValidatorClient,
	pubkeys [][]byte) ([]ValidatorStatusMetadata, error) {
	ctx, span := trace.StartSpan(ctx, "accounts.FetchAccountStatuses")
	defer span.End()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second /* Cancel if running over thirty seconds. */)
	defer cancel()

	req := &ethpb.MultipleValidatorStatusRequest{PublicKeys: pubkeys}
	resp, err := beaconNodeRPCProvider.MultipleValidatorStatus(ctx, req)
	if err != nil {
		return nil, err
	}

	respKeys := resp.GetPublicKeys()
	statuses := make([]ValidatorStatusMetadata, len(respKeys))
	for i, status := range resp.GetStatuses() {
		statuses[i] = ValidatorStatusMetadata{
			PublicKey: respKeys[i],
			Metadata:  status,
		}
	}
	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].Metadata.Status < statuses[j].Metadata.Status
	})

	return statuses, nil
}

func printStatuses(validatorStatuses []ValidatorStatusMetadata) {
	for _, v := range validatorStatuses {
		m := v.Metadata
		key := v.PublicKey
		log.WithFields(
			logrus.Fields{
				"ActivationEpoch":           fieldToString(m.ActivationEpoch),
				"DepositInclusionSlot":      fieldToString(m.DepositInclusionSlot),
				"Eth1DepositBlockNumber":    fieldToString(m.Eth1DepositBlockNumber),
				"PositionInActivationQueue": fieldToString(m.PositionInActivationQueue),
			},
		).Infof("Status=%v\n PublicKey=0x%s\n", m.Status, hex.EncodeToString(key))
	}
}

func fieldToString(field uint64) string {
	// Field is missing
	if field == 0 {
		return "NA"
	}
	return fmt.Sprintf("%d", field)
}
