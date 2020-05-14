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
	"github.com/prysmaticlabs/prysm/shared/keystore"
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

// MaxRequestLimit specifies the max grpc requests allowed
// to fetch account statuses.
const MaxRequestLimit = 5 // XXX: Should create flag to make parameter configurable.

// MaxRequestKeys specifies the max amount of public keys allowed
// in a single grpc request, when fetching account statuses.
const MaxRequestKeys = 2000 // XXX: This is an arbitrary number.
// Should compute max keys allowed before reponse exceeds GrpcMaxCallRecvMsgSizeFlag.

// RunStatusCommand is the entry point to the `validator status` command.
func RunStatusCommand(
	ctx context.Context,
	pubkeys [][]byte,
	withCert string,
	endpoint string,
	maxCallRecvMsgSize int,
	grpcRetries uint,
	grpcHeaders []string) error {
	dialOpts := constructDialOptions(maxCallRecvMsgSize, withCert, grpcHeaders, grpcRetries)
	conn, err := grpc.DialContext(ctx, endpoint, dialOpts...)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed to dial beacon node endpoint at %s", endpoint))
	}
	statuses, err := FetchAccountStatuses(
		ctx, ethpb.NewBeaconNodeValidatorClient(conn), pubkeys)
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
	grpcRetries uint) []grpc.DialOption {
	var transportSecurity grpc.DialOption
	if withCert != "" {
		creds, err := credentials.NewClientTLSFromFile(withCert, "")
		if err != nil {
			log.Errorf("Could not get valid credentials: %v", err)
			return nil
		}
		transportSecurity = grpc.WithTransportCredentials(creds)
	} else {
		transportSecurity = grpc.WithInsecure()
		log.Warn("You are using an insecure gRPC connection! Please provide a certificate and key to use a secure connection.")
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
		grpc.WithTimeout(
			10 * time.Second /* Block for 10 seconds to see if we can connect to beacon node */),
	}
}

// FetchAccountStatuses fetches validator statuses from the BeaconNodeValidatorClient
// for each validator public key.
func FetchAccountStatuses(
	ctx context.Context,
	beaconNodeRPCProvider ethpb.BeaconNodeValidatorClient,
	pubkeys [][]byte) ([]ValidatorStatusMetadata, error) {
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
	// Wait for fetch routines to finish.
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

	return mergeStatuses(allStatuses), err
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

// mergeStatuses merges k sorted ValidatorStatusMetadata slices to 1.
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
