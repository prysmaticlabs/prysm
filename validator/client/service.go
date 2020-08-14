package client

import (
	"context"
	"strings"
	"time"

	"github.com/dgraph-io/ristretto"
	middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	lru "github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/grpcutils"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/validator/db"
	keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v1"
	v2 "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	slashingprotection "github.com/prysmaticlabs/prysm/validator/slashing-protection"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ocgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

var log = logrus.WithField("prefix", "validator")

// ValidatorService represents a service to manage the validator client
// routine.
type ValidatorService struct {
	db                   db.Database
	ctx                  context.Context
	cancel               context.CancelFunc
	validator            Validator
	graffiti             []byte
	conn                 *grpc.ClientConn
	endpoint             string
	withCert             string
	dataDir              string
	keyManager           keymanager.KeyManager
	keyManagerV2         v2.IKeymanager
	logValidatorBalances bool
	emitAccountMetrics   bool
	maxCallRecvMsgSize   int
	validatingPubKeys    [][48]byte
	grpcRetries          uint
	grpcRetryDelay       time.Duration
	grpcHeaders          []string
	protector            slashingprotection.Protector
}

// Config for the validator service.
type Config struct {
	Endpoint                   string
	DataDir                    string
	CertFlag                   string
	GraffitiFlag               string
	ValidatingPubKeys          [][48]byte
	KeyManager                 keymanager.KeyManager
	KeyManagerV2               v2.IKeymanager
	LogValidatorBalances       bool
	EmitAccountMetrics         bool
	GrpcMaxCallRecvMsgSizeFlag int
	GrpcRetriesFlag            uint
	GrpcRetryDelay             time.Duration
	GrpcHeadersFlag            string
	Protector                  slashingprotection.Protector
	ValDB                      db.Database
	Validator                  Validator
}

// NewValidatorService creates a new validator service for the service
// registry.
func NewValidatorService(ctx context.Context, cfg *Config) (*ValidatorService, error) {
	ctx, cancel := context.WithCancel(ctx)
	return &ValidatorService{
		ctx:                  ctx,
		cancel:               cancel,
		endpoint:             cfg.Endpoint,
		withCert:             cfg.CertFlag,
		dataDir:              cfg.DataDir,
		graffiti:             []byte(cfg.GraffitiFlag),
		keyManager:           cfg.KeyManager,
		keyManagerV2:         cfg.KeyManagerV2,
		validatingPubKeys:    cfg.ValidatingPubKeys,
		logValidatorBalances: cfg.LogValidatorBalances,
		emitAccountMetrics:   cfg.EmitAccountMetrics,
		maxCallRecvMsgSize:   cfg.GrpcMaxCallRecvMsgSizeFlag,
		grpcRetries:          cfg.GrpcRetriesFlag,
		grpcRetryDelay:       cfg.GrpcRetryDelay,
		grpcHeaders:          strings.Split(cfg.GrpcHeadersFlag, ","),
		protector:            cfg.Protector,
		validator:            cfg.Validator,
		db:                   cfg.ValDB,
	}, nil
}

// Start the validator service. Launches the main go routine for the validator
// client.
func (v *ValidatorService) Start() {
	streamInterceptor := grpc.WithStreamInterceptor(middleware.ChainStreamClient(
		grpc_opentracing.StreamClientInterceptor(),
		grpc_prometheus.StreamClientInterceptor,
		grpc_retry.StreamClientInterceptor(),
	))
	dialOpts := ConstructDialOptions(
		v.maxCallRecvMsgSize,
		v.withCert,
		v.grpcHeaders,
		v.grpcRetries,
		v.grpcRetryDelay,
		streamInterceptor,
	)
	if dialOpts == nil {
		return
	}
	conn, err := grpc.DialContext(v.ctx, v.endpoint, dialOpts...)
	if err != nil {
		log.Errorf("Could not dial endpoint: %s, %v", v.endpoint, err)
		return
	}
	if v.withCert != "" {
		log.Info("Established secure gRPC connection")
	}

	v.conn = conn
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1920, // number of keys to track.
		MaxCost:     192,  // maximum cost of cache, 1 item = 1 cost.
		BufferItems: 64,   // number of keys per Get buffer.
	})
	if err != nil {
		panic(err)
	}

	aggregatedSlotCommitteeIDCache, err := lru.New(int(params.BeaconConfig().MaxCommitteesPerSlot))
	if err != nil {
		log.Errorf("Could not initialize cache: %v", err)
		return
	}

	v.validator = &validator{
		db:                             v.db,
		validatorClient:                ethpb.NewBeaconNodeValidatorClient(v.conn),
		beaconClient:                   ethpb.NewBeaconChainClient(v.conn),
		node:                           ethpb.NewNodeClient(v.conn),
		keyManager:                     v.keyManager,
		keyManagerV2:                   v.keyManagerV2,
		graffiti:                       v.graffiti,
		logValidatorBalances:           v.logValidatorBalances,
		emitAccountMetrics:             v.emitAccountMetrics,
		startBalances:                  make(map[[48]byte]uint64),
		prevBalance:                    make(map[[48]byte]uint64),
		indexToPubkey:                  make(map[uint64][48]byte),
		pubkeyToIndex:                  make(map[[48]byte]uint64),
		pubkeyToStatus:                 make(map[[48]byte]ethpb.ValidatorStatus),
		attLogs:                        make(map[[32]byte]*attSubmitted),
		domainDataCache:                cache,
		aggregatedSlotCommitteeIDCache: aggregatedSlotCommitteeIDCache,
		protector:                      v.protector,
		voteStats:                      voteStats{startEpoch: ^uint64(0)},
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

// signObject signs a generic object, with protection if available.
// This should only be used for accounts v1.
func (v *validator) signObject(
	ctx context.Context,
	pubKey [48]byte,
	object interface{},
	domain []byte,
) (bls.Signature, error) {
	if featureconfig.Get().EnableAccountsV2 {
		return nil, errors.New("signObject not supported for accounts v2")
	}

	if protectingKeymanager, supported := v.keyManager.(keymanager.ProtectingKeyManager); supported {
		root, err := ssz.HashTreeRoot(object)
		if err != nil {
			return nil, err
		}
		return protectingKeymanager.SignGeneric(pubKey, root, bytesutil.ToBytes32(domain))
	}

	root, err := helpers.ComputeSigningRoot(object, domain)
	if err != nil {
		return nil, err
	}
	return v.keyManager.Sign(pubKey, root)
}

// ConstructDialOptions constructs a list of grpc dial options
func ConstructDialOptions(
	maxCallRecvMsgSize int,
	withCert string,
	grpcHeaders []string,
	grpcRetries uint,
	grpcRetryDelay time.Duration,
	extraOpts ...grpc.DialOption,
) []grpc.DialOption {
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
		log.Warn("You are using an insecure gRPC connection. If you are running your beacon node and " +
			"validator on the same machines, you can ignore this message. If you want to know " +
			"how to enable secure connections, see: https://docs.prylabs.network/docs/prysm-usage/secure-grpc")
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

	dialOpts := []grpc.DialOption{
		transportSecurity,
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxCallRecvMsgSize),
			grpc_retry.WithMax(grpcRetries),
			grpc_retry.WithBackoff(grpc_retry.BackoffLinear(grpcRetryDelay)),
			grpc.Header(&md),
		),
		grpc.WithStatsHandler(&ocgrpc.ClientHandler{}),
		grpc.WithUnaryInterceptor(middleware.ChainUnaryClient(
			grpc_opentracing.UnaryClientInterceptor(),
			grpc_prometheus.UnaryClientInterceptor,
			grpc_retry.UnaryClientInterceptor(),
			grpcutils.LogGRPCRequests,
		)),
		grpc.WithChainStreamInterceptor(
			grpcutils.LogGRPCStream,
			grpc_opentracing.StreamClientInterceptor(),
			grpc_prometheus.StreamClientInterceptor,
			grpc_retry.StreamClientInterceptor(),
		),
	}

	dialOpts = append(dialOpts, extraOpts...)
	return dialOpts
}

// ValidatorBalances returns the validator balances mapping keyed by public keys.
func (v *ValidatorService) ValidatorBalances(ctx context.Context) map[[48]byte]uint64 {
	return v.validator.BalancesByPubkeys(ctx)
}

// ValidatorIndicesToPubkeys returns the validator indices mapping keyed by public keys.
func (v *ValidatorService) ValidatorIndicesToPubkeys(ctx context.Context) map[uint64][48]byte {
	return v.validator.IndicesToPubkeys(ctx)
}

// ValidatorPubkeysToIndices returns the validator public keys mapping keyed by indices.
func (v *ValidatorService) ValidatorPubkeysToIndices(ctx context.Context) map[[48]byte]uint64 {
	return v.validator.PubkeysToIndices(ctx)
}

// ValidatorPubkeysToStatuses returns the validator statuses mapping keyed by public keys.
func (v *ValidatorService) ValidatorPubkeysToStatuses(ctx context.Context) map[[48]byte]ethpb.ValidatorStatus {
	return v.validator.PubkeysToStatuses(ctx)
}
