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
		context.Background(), 10*time.Second /* Cancel if cannot connect to beacon node in 10 seconds. */)
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
