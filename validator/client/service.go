package client

import (
	"context"
	"errors"
	"fmt"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/keystore"
	"github.com/prysmaticlabs/prysm/shared/params"
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
	key                  *keystore.Key
	keys                 map[string]*keystore.Key
	logValidatorBalances bool
}

// Config for the validator service.
type Config struct {
	Endpoint             string
	CertFlag             string
	KeystorePath         string
	Password             string
	LogValidatorBalances bool
}

// NewValidatorService creates a new validator service for the service
// registry.
func NewValidatorService(ctx context.Context, cfg *Config) (*ValidatorService, error) {
	ctx, cancel := context.WithCancel(ctx)
	validatorFolder := cfg.KeystorePath
	validatorPrefix := params.BeaconConfig().ValidatorPrivkeyFileName
	ks := keystore.NewKeystore(cfg.KeystorePath)
	keys, err := ks.GetKeys(validatorFolder, validatorPrefix, cfg.Password)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not get private key: %v", err)
	}
	var key *keystore.Key
	for _, v := range keys {
		key = v
		break
	}
	return &ValidatorService{
		ctx:                  ctx,
		cancel:               cancel,
		endpoint:             cfg.Endpoint,
		withCert:             cfg.CertFlag,
		keys:                 keys,
		key:                  key,
		logValidatorBalances: cfg.LogValidatorBalances,
	}, nil
}

// Start the validator service. Launches the main go routine for the validator
// client.
func (v *ValidatorService) Start() {
	pubkeys := make([][]byte, 0)
	for i := range v.keys {
		log.WithField("publicKey", fmt.Sprintf("%#x", v.keys[i].PublicKey.Marshal())).Info("Initializing new validator service")
		pubkeys = append(pubkeys, v.keys[i].PublicKey.Marshal())
	}

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
	conn, err := grpc.DialContext(v.ctx, v.endpoint, dialOpt, grpc.WithStatsHandler(&ocgrpc.ClientHandler{}))
	if err != nil {
		log.Errorf("Could not dial endpoint: %s, %v", v.endpoint, err)
		return
	}
	log.Info("Successfully started gRPC connection")
	v.conn = conn
	v.validator = &validator{
		beaconClient:         pb.NewBeaconServiceClient(v.conn),
		validatorClient:      pb.NewValidatorServiceClient(v.conn),
		attesterClient:       pb.NewAttesterServiceClient(v.conn),
		proposerClient:       pb.NewProposerServiceClient(v.conn),
		keys:                 v.keys,
		pubkeys:              pubkeys,
		logValidatorBalances: v.logValidatorBalances,
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
