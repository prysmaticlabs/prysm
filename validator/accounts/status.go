package accounts

import (
	"context"
	"fmt"
	"sort"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/keystore"
	"go.opencensus.io/trace"
)

// FetchAccountStatuses fetches validator statuses from the BeaconNodeValidatorClient for each validator public key.
func FetchAccountStatuses(
	ctx context.Context,
	beaconNodeRPCProvider ethpb.BeaconNodeValidatorClient,
	keyPairs map[string]*keystore.Key,
) ([]*ethpb.ValidatorStatusResponse, error) {
	ctx, span := trace.StartSpan(ctx, "validator.FetchAccountStatuses")
	defer span.End()

	var err error
	const RequestLimit = 3
	statuses := make([]*ethpb.ValidatorStatusResponse, 0, len(keyPairs))
	errorChannel := make(chan error, RequestLimit)
	statusChannel := make(chan *ethpb.ValidatorStatusResponse, RequestLimit)

	for _, key := range keyPairs {
		go fetchValidatorStatus(
			ctx, beaconNodeRPCProvider, key.PublicKey.Marshal(), statusChannel, errorChannel)
	}

	for i := 0; i < len(keyPairs); i++ {
		select {
		case status := <-statusChannel:
			statuses = append(statuses, status)
		case e := <-errorChannel:
			err = e
		}
	}

	// Sort responses by status
	// XXX: This sort does not work right now. We need to have the
	// public key of the validator indicated in the response.
	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].Status < statuses[j].Status
	})

	return statuses, err
}

func fetchValidatorStatus(
	ctx context.Context,
	rpcProvder ethpb.BeaconNodeValidatorClient,
	pubKey []byte,
	statusChannel chan *ethpb.ValidatorStatusResponse,
	errorChannel chan error) {
	if ctx.Err() == context.Canceled {
		errorChannel <- errors.Wrap(ctx.Err(), "context has been canceled.")
		return
	}

	req := &ethpb.ValidatorStatusRequest{
		PublicKey: pubKey,
	}
	status, err := rpcProvder.ValidatorStatus(ctx, req)
	if err != nil {
		errorChannel <- errors.Wrap(
			err,
			fmt.Sprintf("could not fetch validator status for %v from the BeaconNodeValidatorClient", pubKey))
		return
	}
	statusChannel <- status
}
