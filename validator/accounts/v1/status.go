package accounts

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// statusTimeout defines a period after which request to fetch account statuses is cancelled.
const statusTimeout = 30 * time.Second

// ValidatorStatusMetadata holds all status information about a validator.
type ValidatorStatusMetadata struct {
	PublicKey []byte
	Index     uint64
	Metadata  *ethpb.ValidatorStatusResponse
}

// RunStatusCommand is the entry point to the `validator status` command.
func RunStatusCommand(pubKeys [][]byte, beaconNodeRPCProvider ethpb.BeaconNodeValidatorClient) error {
	statuses, err := FetchAccountStatuses(context.Background(), beaconNodeRPCProvider, pubKeys)
	if err != nil {
		return errors.Wrap(err, "could not fetch account statuses from the beacon node")
	}
	printStatuses(statuses)
	return nil
}

// FetchAccountStatuses fetches validator statuses from the BeaconNodeValidatorClient
// for each validator public key.
func FetchAccountStatuses(
	ctx context.Context,
	beaconClient ethpb.BeaconNodeValidatorClient,
	pubKeys [][]byte,
) ([]ValidatorStatusMetadata, error) {
	ctx, span := trace.StartSpan(ctx, "accounts.FetchAccountStatuses")
	defer span.End()
	ctx, cancel := context.WithTimeout(ctx, statusTimeout)
	defer cancel()

	req := &ethpb.MultipleValidatorStatusRequest{PublicKeys: pubKeys}
	resp, err := beaconClient.MultipleValidatorStatus(ctx, req)
	if err != nil {
		return nil, err
	}

	statuses := make([]ValidatorStatusMetadata, len(resp.Statuses))
	for i, status := range resp.Statuses {
		statuses[i] = ValidatorStatusMetadata{
			PublicKey: resp.PublicKeys[i],
			Index:     resp.Indices[i],
			Metadata:  status,
		}
	}
	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].Metadata.Status < statuses[j].Metadata.Status
	})

	return statuses, nil
}

func printStatuses(validatorStatuses []ValidatorStatusMetadata) {
	nonexistentIndex := ^uint64(0)
	for _, v := range validatorStatuses {
		m := v.Metadata
		key := v.PublicKey
		fields := logrus.Fields{
			"publicKey": fmt.Sprintf("%#x", key),
		}
		if v.Index != nonexistentIndex {
			fields["index"] = v.Index
		}
		if m.Status == ethpb.ValidatorStatus_PENDING || m.Status == ethpb.ValidatorStatus_ACTIVE {
			fields["activationEpoch"] = m.ActivationEpoch
			if m.ActivationEpoch == params.BeaconConfig().FarFutureEpoch {
				fields["positionInActivationQueue"] = m.PositionInActivationQueue
			}
		} else if m.Status == ethpb.ValidatorStatus_DEPOSITED {
			if m.PositionInActivationQueue != 0 {
				fields["depositInclusionSlot"] = m.DepositInclusionSlot
				fields["eth1DepositBlockNumber"] = m.Eth1DepositBlockNumber
			} else {
				fields["positionInActivationQueue"] = m.PositionInActivationQueue
			}
		}
		log.WithFields(fields).Infof("Status: %s", m.Status.String())
	}
}
