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
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	grpcutil "github.com/prysmaticlabs/prysm/api/grpc"
	"github.com/prysmaticlabs/prysm/async/event"
	lruwrpr "github.com/prysmaticlabs/prysm/cache/lru"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	validator_service_config "github.com/prysmaticlabs/prysm/config/validator/service"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/client/iface"
	"github.com/prysmaticlabs/prysm/validator/db"
	"github.com/prysmaticlabs/prysm/validator/graffiti"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/local"
	remote_web3signer "github.com/prysmaticlabs/prysm/validator/keymanager/remote-web3signer"
	"go.opencensus.io/plugin/ocgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/types/known/emptypb"
)

// SyncChecker is able to determine if a beacon node is currently
// going through chain synchronization.
type SyncChecker interface {
	Syncing(ctx context.Context) (bool, error)
}

// GenesisFetcher can retrieve genesis information such as
// the genesis time and the validator deposit contract address.
type GenesisFetcher interface {
	GenesisInfo(ctx context.Context) (*ethpb.Genesis, error)
}

// ValidatorService represents a service to manage the validator client
// routine.
type ValidatorService struct {
	useWeb                bool
	emitAccountMetrics    bool
	logValidatorBalances  bool
	logDutyCountDown      bool
	interopKeysConfig     *local.InteropKeymanagerConfig
	conn                  *grpc.ClientConn
	grpcRetryDelay        time.Duration
	grpcRetries           uint
	maxCallRecvMsgSize    int
	cancel                context.CancelFunc
	walletInitializedFeed *event.Feed
	wallet                *wallet.Wallet
	graffitiStruct        *graffiti.Graffiti
	dataDir               string
	withCert              string
	endpoint              string
	ctx                   context.Context
	validator             iface.Validator
	db                    db.Database
	grpcHeaders           []string
	graffiti              []byte
	web3SignerConfig      *remote_web3signer.SetupConfig
	feeRecipientConfig    *validator_service_config.FeeRecipientConfig
}

// Config for the validator service.
type Config struct {
	UseWeb                     bool
	LogValidatorBalances       bool
	EmitAccountMetrics         bool
	LogDutyCountDown           bool
	InteropKeysConfig          *local.InteropKeymanagerConfig
	Wallet                     *wallet.Wallet
	WalletInitializedFeed      *event.Feed
	GrpcRetriesFlag            uint
	GrpcMaxCallRecvMsgSizeFlag int
	GrpcRetryDelay             time.Duration
	GraffitiStruct             *graffiti.Graffiti
	Validator                  iface.Validator
	ValDB                      db.Database
	CertFlag                   string
	DataDir                    string
	GrpcHeadersFlag            string
	GraffitiFlag               string
	Endpoint                   string
	Web3SignerConfig           *remote_web3signer.SetupConfig
	FeeRecipientConfig         *validator_service_config.FeeRecipientConfig
}

// NewValidatorService creates a new validator service for the service
// registry.
func NewValidatorService(ctx context.Context, cfg *Config) (*ValidatorService, error) {
	ctx, cancel := context.WithCancel(ctx)
	return &ValidatorService{
		ctx:                   ctx,
		cancel:                cancel,
		endpoint:              cfg.Endpoint,
		withCert:              cfg.CertFlag,
		dataDir:               cfg.DataDir,
		graffiti:              []byte(cfg.GraffitiFlag),
		logValidatorBalances:  cfg.LogValidatorBalances,
		emitAccountMetrics:    cfg.EmitAccountMetrics,
		maxCallRecvMsgSize:    cfg.GrpcMaxCallRecvMsgSizeFlag,
		grpcRetries:           cfg.GrpcRetriesFlag,
		grpcRetryDelay:        cfg.GrpcRetryDelay,
		grpcHeaders:           strings.Split(cfg.GrpcHeadersFlag, ","),
		validator:             cfg.Validator,
		db:                    cfg.ValDB,
		wallet:                cfg.Wallet,
		walletInitializedFeed: cfg.WalletInitializedFeed,
		useWeb:                cfg.UseWeb,
		interopKeysConfig:     cfg.InteropKeysConfig,
		graffitiStruct:        cfg.GraffitiStruct,
		logDutyCountDown:      cfg.LogDutyCountDown,
		web3SignerConfig:      cfg.Web3SignerConfig,
		feeRecipientConfig:    cfg.FeeRecipientConfig,
	}, nil
}

// Start the validator service. Launches the main go routine for the validator
// client.
func (v *ValidatorService) Start() {
	dialOpts := ConstructDialOptions(
		v.maxCallRecvMsgSize,
		v.withCert,
		v.grpcRetries,
		v.grpcRetryDelay,
	)
	if dialOpts == nil {
		return
	}

	v.ctx = grpcutil.AppendHeaders(v.ctx, v.grpcHeaders)

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

	aggregatedSlotCommitteeIDCache := lruwrpr.New(int(params.BeaconConfig().MaxCommitteesPerSlot))

	sPubKeys, err := v.db.EIPImportBlacklistedPublicKeys(v.ctx)
	if err != nil {
		log.Errorf("Could not read slashable public keys from disk: %v", err)
		return
	}
	slashablePublicKeys := make(map[[fieldparams.BLSPubkeyLength]byte]bool)
	for _, pubKey := range sPubKeys {
		slashablePublicKeys[pubKey] = true
	}

	graffitiOrderedIndex, err := v.db.GraffitiOrderedIndex(v.ctx, v.graffitiStruct.Hash)
	if err != nil {
		log.Errorf("Could not read graffiti ordered index from disk: %v", err)
		return
	}

	valStruct := &validator{
		db:                             v.db,
		validatorClient:                ethpb.NewBeaconNodeValidatorClient(v.conn),
		beaconClient:                   ethpb.NewBeaconChainClient(v.conn),
		slashingProtectionClient:       ethpb.NewSlasherClient(v.conn),
		node:                           ethpb.NewNodeClient(v.conn),
		graffiti:                       v.graffiti,
		logValidatorBalances:           v.logValidatorBalances,
		emitAccountMetrics:             v.emitAccountMetrics,
		startBalances:                  make(map[[fieldparams.BLSPubkeyLength]byte]uint64),
		prevBalance:                    make(map[[fieldparams.BLSPubkeyLength]byte]uint64),
		pubkeyToValidatorIndex:         make(map[[fieldparams.BLSPubkeyLength]byte]types.ValidatorIndex),
		attLogs:                        make(map[[32]byte]*attSubmitted),
		domainDataCache:                cache,
		aggregatedSlotCommitteeIDCache: aggregatedSlotCommitteeIDCache,
		voteStats:                      voteStats{startEpoch: types.Epoch(^uint64(0))},
		useWeb:                         v.useWeb,
		interopKeysConfig:              v.interopKeysConfig,
		wallet:                         v.wallet,
		walletInitializedFeed:          v.walletInitializedFeed,
		blockFeed:                      new(event.Feed),
		graffitiStruct:                 v.graffitiStruct,
		graffitiOrderedIndex:           graffitiOrderedIndex,
		eipImportBlacklistedPublicKeys: slashablePublicKeys,
		logDutyCountDown:               v.logDutyCountDown,
		Web3SignerConfig:               v.web3SignerConfig,
		feeRecipientConfig:             v.feeRecipientConfig,
		walletIntializedChannel:        make(chan *wallet.Wallet, 1),
	}
	// To resolve a race condition at startup due to the interface
	// nature of the abstracted block type. We initialize
	// the inner type of the feed before hand. So that
	// during future accesses, there will be no panics here
	// from type incompatibility.
	tempChan := make(chan block.SignedBeaconBlock)
	sub := valStruct.blockFeed.Subscribe(tempChan)
	sub.Unsubscribe()
	close(tempChan)

	v.validator = valStruct
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

// Status of the validator service.
func (v *ValidatorService) Status() error {
	if v.conn == nil {
		return errors.New("no connection to beacon RPC")
	}
	return nil
}

// UseInteropKeys returns the useInteropKeys flag.
func (v *ValidatorService) InteropKeysConfig() *local.InteropKeymanagerConfig {
	return v.interopKeysConfig
}

func (v *ValidatorService) Keymanager() (keymanager.IKeymanager, error) {
	return v.validator.Keymanager()
}

// ConstructDialOptions constructs a list of grpc dial options
func ConstructDialOptions(
	maxCallRecvMsgSize int,
	withCert string,
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

	dialOpts := []grpc.DialOption{
		transportSecurity,
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxCallRecvMsgSize),
			grpc_retry.WithMax(grpcRetries),
			grpc_retry.WithBackoff(grpc_retry.BackoffLinear(grpcRetryDelay)),
		),
		grpc.WithStatsHandler(&ocgrpc.ClientHandler{}),
		grpc.WithUnaryInterceptor(middleware.ChainUnaryClient(
			grpc_opentracing.UnaryClientInterceptor(),
			grpc_prometheus.UnaryClientInterceptor,
			grpc_retry.UnaryClientInterceptor(),
			grpcutil.LogRequests,
		)),
		grpc.WithChainStreamInterceptor(
			grpcutil.LogStream,
			grpc_opentracing.StreamClientInterceptor(),
			grpc_prometheus.StreamClientInterceptor,
			grpc_retry.StreamClientInterceptor(),
		),
		grpc.WithResolvers(&multipleEndpointsGrpcResolverBuilder{}),
	}

	dialOpts = append(dialOpts, extraOpts...)
	return dialOpts
}

// Syncing returns whether or not the beacon node is currently synchronizing the chain.
func (v *ValidatorService) Syncing(ctx context.Context) (bool, error) {
	nc := ethpb.NewNodeClient(v.conn)
	resp, err := nc.GetSyncStatus(ctx, &emptypb.Empty{})
	if err != nil {
		return false, err
	}
	return resp.Syncing, nil
}

// GenesisInfo queries the beacon node for the chain genesis info containing
// the genesis time along with the validator deposit contract address.
func (v *ValidatorService) GenesisInfo(ctx context.Context) (*ethpb.Genesis, error) {
	nc := ethpb.NewNodeClient(v.conn)
	return nc.GetGenesis(ctx, &emptypb.Empty{})
}
