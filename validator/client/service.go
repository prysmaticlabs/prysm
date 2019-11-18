package client

import (
	"context"
	"fmt"

	middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/pkg/errors"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/keystore"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ocgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var log = logrus.WithField("prefix", "validator")

// ValidatorService represents a service to manage the validator client
// routine.
type ValidatorService struct {
	ctx                  context.Context
	cancel               context.CancelFunc
	validator            Validator
	conn                 *grpc.ClientConn
	endpoint             string
	withCert             string
	keys                 map[[48]byte]*keystore.Key
	logValidatorBalances bool
}

// Config for the validator service.
type Config struct {
	Endpoint             string
	CertFlag             string
	Keys                 map[string]*keystore.Key
	LogValidatorBalances bool
}

// NewValidatorService creates a new validator service for the service
// registry.
func NewValidatorService(ctx context.Context, cfg *Config) (*ValidatorService, error) {
	ctx, cancel := context.WithCancel(ctx)
	pubKeys := make(map[[48]byte]*keystore.Key)
	for _, v := range cfg.Keys {
		var pubKey [48]byte
		copy(pubKey[:], v.PublicKey.Marshal())
		pubKeys[pubKey] = v
	}
	return &ValidatorService{
		ctx:                  ctx,
		cancel:               cancel,
		endpoint:             cfg.Endpoint,
		withCert:             cfg.CertFlag,
		keys:                 pubKeys,
		logValidatorBalances: cfg.LogValidatorBalances,
	}, nil
}

// Start the validator service. Launches the main go routine for the validator
// client.
func (v *ValidatorService) Start() {
	pubKeys := pubKeysFromMap(v.keys)

	var dialOpt grpc.DialOption
	if v.withCert != "" {
		creds, err := credentials.NewClientTLSFromFile(v.withCert, "")
		if err != nil {
			log.Errorf("Could not get valid credentials: %v", err)
			return
		}
		dialOpt = grpc.WithTransportCredentials(creds)
	} else {
		dialOpt = grpc.WithInsecure()
		log.Warn("You are using an insecure gRPC connection! Please provide a certificate and key to use a secure connection.")
	}
	opts := []grpc.DialOption{
		dialOpt,
		grpc.WithStatsHandler(&ocgrpc.ClientHandler{}),
		grpc.WithStreamInterceptor(middleware.ChainStreamClient(
			grpc_opentracing.StreamClientInterceptor(),
			grpc_prometheus.StreamClientInterceptor,
		)),
		grpc.WithUnaryInterceptor(middleware.ChainUnaryClient(
			grpc_opentracing.UnaryClientInterceptor(),
			grpc_prometheus.UnaryClientInterceptor,
		)),
	}
	conn, err := grpc.DialContext(v.ctx, v.endpoint, opts...)
	if err != nil {
		log.Errorf("Could not dial endpoint: %s, %v", v.endpoint, err)
		return
	}
	log.Info("Successfully started gRPC connection")
	v.conn = conn
	v.validator = &validator{
		validatorClient:      pb.NewValidatorServiceClient(v.conn),
		attesterClient:       pb.NewAttesterServiceClient(v.conn),
		proposerClient:       pb.NewProposerServiceClient(v.conn),
		node:                 ethpb.NewNodeClient(v.conn),
		keys:                 v.keys,
		pubkeys:              pubKeys,
		logValidatorBalances: v.logValidatorBalances,
		prevBalance:          make(map[[48]byte]uint64),
	}
	go run(v.ctx, v.validator)
}

// Stop the validator service.
func (v *ValidatorService) Stop() error {
	v.cancel()
	log.Info("Stopping service")
	if v.conn != nil {
		return v.conn.Close()
	}
	return nil
}

// Status ...
//
// WIP - not done.
func (v *ValidatorService) Status() error {
	if v.conn == nil {
		return errors.New("no connection to beacon RPC")
	}
	return nil
}

// pubKeysFromMap is a helper that creates an array of public keys given a map of keystores
func pubKeysFromMap(keys map[[48]byte]*keystore.Key) [][]byte {
	pubKeys := make([][]byte, 0)
	for pubKey := range keys {
		var pubKeyCopy [48]byte
		copy(pubKeyCopy[:], pubKey[:])
		pubKeys = append(pubKeys, pubKeyCopy[:])
		log.WithField("pubKey", fmt.Sprintf("%#x", bytesutil.Trunc(pubKeyCopy[:]))).Info("New validator service")
	}
	return pubKeys
}
