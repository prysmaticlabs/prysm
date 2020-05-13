package accounts

import (
	"context"
	"encoding/hex"
	"fmt"
	"sort"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/keystore"
	"go.opencensus.io/trace"
)

// ValidatorStatusMetadata holds all status information about a validator.
type ValidatorStatusMetadata struct {
	PublicKey []byte
	Metadata  *ethpb.ValidatorStatusResponse
}

// MaxRequestLimit specifies the max grpc requests allowed
// to fetch account statuses.
const MaxRequestLimit = 3

// MaxRequestKeys specifies the max amount of public keys allowed
// in a single grpc request, when fetching account statuses.
const MaxRequestKeys = 2000 // XXX: This is an arbitrary number.
// Should compute max keys allowed before exceeding GrpcMaxCallRecvMsgSizeFlag

// FetchAccountStatuses fetches validator statuses from the BeaconNodeValidatorClient
// for each validator public key.
func FetchAccountStatuses(
	ctx context.Context,
	beaconNodeRPCProvider ethpb.BeaconNodeValidatorClient,
	pubkeys [][]byte) ([][]ValidatorStatusMetadata, error) {
	ctx, span := trace.StartSpan(ctx, "validator.FetchAccountStatuses")
	defer span.End()

	errorChannel := make(chan error, MaxRequestLimit)
	statusChannel := make(chan []ValidatorStatusMetadata, MaxRequestLimit)
	// Launch routines to fetch statuses.
	i, numBatches := 0, 0
	for ; i+MaxRequestKeys < len(pubkeys); i += MaxRequestKeys {
		go fetchValidatorStatus(
			ctx, beaconNodeRPCProvider, pubkeys[i:i+MaxRequestKeys], statusChannel, errorChannel)
		numBatches++
	}
	if i < len(pubkeys) {
		go fetchValidatorStatus(
			ctx, beaconNodeRPCProvider, pubkeys[i:], statusChannel, errorChannel)
		numBatches++
	}
	// Wait from fetch routines to finish.
	var err error
	allStatuses := make([][]ValidatorStatusMetadata, 0, numBatches)
	for i := 0; i < numBatches; i++ {
		select {
		case statuses := <-statusChannel:
			allStatuses = append(allStatuses, statuses)
		case e := <-errorChannel:
			err = e
		}
	}

	return allStatuses, err
}

func fetchValidatorStatus(
	ctx context.Context,
	rpcProvder ethpb.BeaconNodeValidatorClient,
	pubkeys [][]byte,
	statusChannel chan []ValidatorStatusMetadata,
	errorChannel chan error) {
	if ctx.Err() == context.Canceled {
		errorChannel <- errors.Wrap(ctx.Err(), "context has been canceled.")
		return
	}

	req := &ethpb.MultipleValidatorStatusRequest{PublicKeys: pubkeys}
	resp, err := rpcProvder.MultipleValidatorStatus(ctx, req)
	if err != nil {
		errorChannel <- errors.Wrap(
			err,
			fmt.Sprintf("Failed to fetch validator statuses for %d key(s)", len(pubkeys)))
		return
	}

	statusChannel <- responseToSortedMetadata(resp)
}

func responseToSortedMetadata(resp *ethpb.MultipleValidatorStatusResponse) []ValidatorStatusMetadata {
	pubkeys := resp.GetPublicKeys()
	validatorStatuses := make([]ValidatorStatusMetadata, len(pubkeys))
	for i, status := range resp.GetStatuses() {
		validatorStatuses[i] = ValidatorStatusMetadata{
			PublicKey: pubkeys[i],
			Metadata:  status,
		}
	}
	sort.Slice(validatorStatuses, func(i, j int) bool {
		return validatorStatuses[i].Metadata.Status < validatorStatuses[j].Metadata.Status
	})
	return validatorStatuses
}

// ExtractPublicKeys extracts only the public keys from the decrypted keys from the keystore.
func ExtractPublicKeys(decryptedKeys map[string]*keystore.Key) [][]byte {
	i := 0
	pubkeys := make([][]byte, len(decryptedKeys))
	for _, key := range decryptedKeys {
		pubkeys[i] = key.PublicKey.Marshal()
		i++
	}
	return pubkeys
}

// MergeStatuses merges k sorted ValidatorStatusMetadata slices to 1.
func MergeStatuses(allStatuses [][]ValidatorStatusMetadata) []ValidatorStatusMetadata {
	if len(allStatuses) == 0 {
		return []ValidatorStatusMetadata{}
	}
	if len(allStatuses) == 1 {
		return allStatuses[0]
	}
	leftHalf := allStatuses[:len(allStatuses)/2]
	rightHalf := allStatuses[len(allStatuses)/2:]
	return mergeTwo(MergeStatuses(leftHalf), MergeStatuses(rightHalf))
}

func mergeTwo(s1, s2 []ValidatorStatusMetadata) []ValidatorStatusMetadata {
	i, j, k := 0, 0, 0
	sortedStatuses := make([]ValidatorStatusMetadata, len(s1)+len(s2))
	for j < len(s1) && k < len(s2) {
		if s1[j].Metadata.Status < s2[k].Metadata.Status {
			sortedStatuses[i] = s1[j]
			j++
		} else {
			sortedStatuses[i] = s2[k]
			k++
		}
		i++
	}
	for j < len(s1) {
		sortedStatuses[i] = s1[j]
		i, j = i+1, j+1
	}
	for k < len(s2) {
		sortedStatuses[i] = s2[k]
		i, k = i+1, k+1
	}
	return sortedStatuses
}

// PrintValidatorStatusMetadata prints out validator statuses and its corresponding metadata
func PrintValidatorStatusMetadata(validatorStatuses []ValidatorStatusMetadata) {
	for _, v := range validatorStatuses {
		m := v.Metadata
		key := v.PublicKey
		fmt.Printf(
			"ValidatorKey: 0x%s, Status: %v\n", hex.EncodeToString(key), m.Status)
		fmt.Printf(
			"Eth1DepositBlockNumber: %s, DepositInclusionSlot: %s, ",
			fieldToString(m.Eth1DepositBlockNumber), fieldToString(m.DepositInclusionSlot))
		fmt.Printf(
			"ActivationEpoch: %s, PositionInActivationQueue: %s\n",
			fieldToString(m.ActivationEpoch), fieldToString(m.PositionInActivationQueue))
	}
}

func fieldToString(field uint64) string {
	// Field is missing
	if field == 0 {
		return "NA"
	}
	return fmt.Sprintf("%d", field)
}
