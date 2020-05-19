package accounts

import (
	"context"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

// ValidatorStatusMetadata holds all status information about a validator.
type ValidatorStatusMetadata struct {
	PublicKey []byte
	Metadata  *ethpb.ValidatorStatusResponse
}

// MaxRequestLimit specifies the max concurrent grpc requests allowed
// to fetch account statuses.
const MaxRequestLimit = 5 // XXX: Should create flag to make parameter configurable.

// MaxRequestKeys specifies the max amount of public keys allowed
// in a single grpc request, when fetching account statuses.
const MaxRequestKeys = 2000 // XXX: This is an arbitrary number. Used to limit time complexity
// of sorting a single batch of status requests. Do not make this number too big.

// RunStatusCommand is the entry point to the `validator status` command.
func RunStatusCommand(
	pubkeys [][]byte,
	withCert string,
	endpoint string,
	maxCallRecvMsgSize int,
	grpcRetries uint,
	grpcHeaders []string) error {
	dialOpts, err := constructDialOptions(maxCallRecvMsgSize, withCert, grpcHeaders, grpcRetries)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(
		context.Background(), 10 * time.Second /* Cancel if cannot connect to beacon node in 10 seconds. */)
	defer cancel()
	conn, err := grpc.DialContext(ctx, endpoint, dialOpts...)
	if err != nil {
		return errors.Wrapf(err, "Failed to dial beacon node endpoint at %s", endpoint)
	}
	statuses, err := FetchAccountStatuses(
		context.Background(), ethpb.NewBeaconNodeValidatorClient(conn), pubkeys)
	if e := conn.Close(); e != nil {
		log.WithError(e).Error("Could not close connection to beacon node")
	}
	if err != nil {
		return errors.Wrap(err, "Could not fetch account statuses from the beacon node")
	}
	printStatuses(statuses)
	return nil
}

func constructDialOptions(
	maxCallRecvMsgSize int,
	withCert string,
	grpcHeaders []string,
	grpcRetries uint) ([]grpc.DialOption, error) {
	var transportSecurity grpc.DialOption
	if withCert != "" {
		creds, err := credentials.NewClientTLSFromFile(withCert, "")
		if err != nil {
			return nil, errors.Wrapf(err, "Could not get valid credentials: %v", err)
		}
		transportSecurity = grpc.WithTransportCredentials(creds)
	} else {
		transportSecurity = grpc.WithInsecure()
		log.Warn(
			"You are using an insecure gRPC connection! Please provide a certificate and key to use a secure connection.")
	}

	if maxCallRecvMsgSize == 0 {
		maxCallRecvMsgSize = 10 * 5 << 20 // Default 50Mb
	}

	md := make(metadata.MD)
	for _, hdr := range grpcHeaders {
		if hdr != "" {
			ss := strings.Split(hdr, "=")
			if len(ss) != 2 {
				log.Warnf("Incorrect gRPC header flag format. Skipping %v", hdr)
				continue
			}
			md.Set(ss[0], ss[1])
		}
	}

	return []grpc.DialOption{
		transportSecurity,
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxCallRecvMsgSize),
			grpc_retry.WithMax(grpcRetries),
			grpc.Header(&md),
		),
		grpc.WithStatsHandler(&ocgrpc.ClientHandler{}),
		grpc.WithBlock(),
	}, nil
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
	// Wait for fetch routines to finish.
	var err error
	allStatuses := make([][]ValidatorStatusMetadata, 0, numBatches)
	for i := 0; i < numBatches; i++ {
		select {
		case statuses := <-statusChannel:
			allStatuses = append(allStatuses, statuses)
		case e := <-errorChannel:
			err = e
			log.Warnln(err)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if err != nil {
		return nil, err
	}
	return mergeStatuses(allStatuses), nil
}

func fetchValidatorStatus(
	ctx context.Context,
	rpcProvider ethpb.BeaconNodeValidatorClient,
	pubkeys [][]byte,
	statusChannel chan []ValidatorStatusMetadata,
	errorChannel chan error) {
	if ctx.Err() == context.Canceled {
		errorChannel <- errors.Wrap(ctx.Err(), "context has been canceled.")
		return
	}

	req := &ethpb.MultipleValidatorStatusRequest{PublicKeys: pubkeys}
	resp, err := rpcProvider.MultipleValidatorStatus(ctx, req)
	if err != nil {
		errorChannel <- errors.Wrapf(
			err, "Failed to fetch validator statuses for %d key(s)", len(pubkeys))
		return
	}
	// Convert response to ValidatorStatusMetadata and sort.
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

	statusChannel <- statuses
}

// mergeStatuses merges k sorted ValidatorStatusMetadata slices to 1.
// XXX: This function should run in O(nlogk) time. For better performance, fix mergeTwo.
func mergeStatuses(allStatuses [][]ValidatorStatusMetadata) []ValidatorStatusMetadata {
	if len(allStatuses) == 0 {
		return []ValidatorStatusMetadata{}
	}
	if len(allStatuses) == 1 {
		return allStatuses[0]
	}
	leftHalf := allStatuses[:len(allStatuses)/2]
	rightHalf := allStatuses[len(allStatuses)/2:]
	return mergeTwo(mergeStatuses(leftHalf), mergeStatuses(rightHalf))
}

// mergeTwo merges two sorted ValidatorStatusMetadata arrays to 1.
// XXX: This function can be improved to run in linear time.
func mergeTwo(s1, s2 []ValidatorStatusMetadata) []ValidatorStatusMetadata {
	statuses := []ValidatorStatusMetadata{}
	statuses = append(statuses, s1...)
	statuses = append(statuses, s2...)
	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].Metadata.Status < statuses[j].Metadata.Status
	})
	return statuses
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
