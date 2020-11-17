package slashingprotection

import (
	"context"
	"fmt"
	"strings"

	middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/grpcutils"
	"go.opencensus.io/plugin/ocgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

// RemoteProtector --
type RemoteProtector struct {
	ctx             context.Context
	slasherEndpoint string
	slasherClient   slashpb.SlasherClient
	conn            *grpc.ClientConn
	opts            []grpc.DialOption
}

func NewRemoteProtector(ctx context.Context, config *Config) (*RemoteProtector, error) {
	var dialOpt grpc.DialOption
	if config.CertFlag != "" {
		creds, err := credentials.NewClientTLSFromFile(config.CertFlag, "")
		if err != nil {
			return nil, errors.Wrap(err, "could not get valid slasher credentials")
		}
		dialOpt = grpc.WithTransportCredentials(creds)
	} else {
		dialOpt = grpc.WithInsecure()
		log.Warn("You are using an insecure slasher gRPC connection! Please provide a certificate and key to use a secure connection.")
	}

	md := make(metadata.MD)
	for _, hdr := range strings.Split(config.GrpcHeadersFlag, ",") {
		if hdr != "" {
			ss := strings.Split(hdr, "=")
			if len(ss) != 2 {
				log.Warnf("Incorrect gRPC header flag format. Skipping %v", hdr)
				continue
			}
			md.Set(ss[0], ss[1])
		}
	}

	opts := []grpc.DialOption{
		dialOpt,
		grpc.WithDefaultCallOptions(
			grpc_retry.WithMax(config.GrpcRetriesFlag),
			grpc_retry.WithBackoff(grpc_retry.BackoffLinear(config.GrpcRetryDelay)),
			grpc.Header(&md),
		),
		grpc.WithStatsHandler(&ocgrpc.ClientHandler{}),
		grpc.WithStreamInterceptor(middleware.ChainStreamClient(
			grpc_opentracing.StreamClientInterceptor(),
			grpc_prometheus.StreamClientInterceptor,
			grpc_retry.StreamClientInterceptor(),
		)),
		grpc.WithUnaryInterceptor(middleware.ChainUnaryClient(
			grpc_opentracing.UnaryClientInterceptor(),
			grpc_prometheus.UnaryClientInterceptor,
			grpc_retry.UnaryClientInterceptor(),
			grpcutils.LogGRPCRequests,
		)),
	}
	return &RemoteProtector{
		ctx:             ctx,
		slasherEndpoint: config.SlasherEndpoint,
		opts:            opts,
	}, nil
}

// Start the remote protector service.
func (rp *RemoteProtector) Start() {
	conn, err := grpc.DialContext(rp.ctx, rp.slasherEndpoint, rp.opts...)
	if err != nil {
		log.Errorf("Could not dial slasher endpoint: %s", rp.slasherEndpoint)
		return
	}
	rp.slasherClient = slashpb.NewSlasherClient(conn)
	log.Debug("Successfully started slasher gRPC connection")
}

// Stop the remote protector service.
func (rp *RemoteProtector) Stop() error {
	log.Info("Stopping remote slashing protector")
	if rp.conn != nil {
		return rp.conn.Close()
	}
	return nil
}

// Status checks if the connection to slasher server is ready,
// returns error otherwise.
func (rp *RemoteProtector) Status() error {
	if rp.conn == nil {
		return errors.New("no connection to slasher RPC")
	}
	if rp.conn.GetState() != connectivity.Ready {
		return fmt.Errorf(
			"cannot connect to slasher server at: %v connection status: %v",
			rp.slasherEndpoint,
			rp.conn.GetState(),
		)
	}
	return nil
}

// CheckBlockSafety this function is part of slashing protection for block proposals it performs
// validation without db update. To be used before the block is signed.
func (rp *RemoteProtector) IsSlashableBlock(ctx context.Context, blockHeader *ethpb.BeaconBlockHeader) bool {
	slashable, err := rp.slasherClient.IsSlashableBlockNoUpdate(ctx, blockHeader)
	if err != nil {
		log.Errorf("External slashing block protection returned an error: %v", err)
		return false
	}
	if slashable != nil && slashable.Slashable {
		log.Warn("External slashing proposal protection found the block to be slashable")
	}
	return !slashable.Slashable
}

// CommitBlock this function is part of slashing protection for block proposals it performs
// validation and db update. To be used after the block is proposed.
func (rp *RemoteProtector) CommitBlock(ctx context.Context, blockHeader *ethpb.SignedBeaconBlockHeader) (bool, error) {
	ps, err := rp.slasherClient.IsSlashableBlock(ctx, blockHeader)
	if err != nil {
		log.Errorf("External slashing block protection returned an error: %v", err)
		return false, err
	}
	if ps != nil && ps.ProposerSlashing != nil {
		log.Warn("External slashing proposal protection found the block to be slashable")
		return false, nil
	}
	return true, nil
}

// CheckAttestationSafety implements the slashing protection for attestations without db update.
// To be used before signing.
func (rp *RemoteProtector) IsSlashableAttestation(ctx context.Context, attestation *ethpb.IndexedAttestation) bool {
	slashable, err := rp.slasherClient.IsSlashableAttestationNoUpdate(ctx, attestation)
	if err != nil {
		log.Errorf("External slashing attestation protection returned an error: %v", err)
		return false
	}
	if slashable.Slashable {
		log.Warn("External slashing attestation protection found the attestation to be slashable")
	}
	return !slashable.Slashable
}

// CommitAttestation implements the slashing protection for attestations it performs
// validation and db update. To be used after the attestation is proposed.
func (rp *RemoteProtector) CommitAttestation(ctx context.Context, attestation *ethpb.IndexedAttestation) bool {
	as, err := rp.slasherClient.IsSlashableAttestation(ctx, attestation)
	if err != nil {
		log.Errorf("External slashing attestation protection returned an error: %v", err)
		return false
	}
	if as != nil && as.AttesterSlashing != nil {
		log.Warnf("External slashing attestation protection found the attestation to be slashable: %v", as)
		return false
	}
	return true
}
