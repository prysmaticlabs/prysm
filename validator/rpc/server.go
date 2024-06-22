package rpc

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/api"
	"github.com/prysmaticlabs/prysm/v5/api/server/httprest"
	"github.com/prysmaticlabs/prysm/v5/async/event"
	"github.com/prysmaticlabs/prysm/v5/io/logs"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/v5/validator/client"
	iface "github.com/prysmaticlabs/prysm/v5/validator/client/iface"
	"github.com/prysmaticlabs/prysm/v5/validator/db"
	"github.com/prysmaticlabs/prysm/v5/validator/web"
)

// Config options for the HTTP server.
type Config struct {
	Host                   string // http host, not grpc
	Port                   int    // http port, not grpc
	GRPCMaxCallRecvMsgSize int
	GRPCRetries            uint
	GRPCRetryDelay         time.Duration
	GRPCHeaders            []string
	BeaconNodeGRPCEndpoint string
	BeaconApiEndpoint      string
	BeaconApiTimeout       time.Duration
	BeaconNodeCert         string
	DB                     db.Database
	Wallet                 *wallet.Wallet
	WalletDir              string
	WalletInitializedFeed  *event.Feed
	ValidatorService       *client.ValidatorService
	AuthTokenPath          string
	Router                 *mux.Router
}

// Server defining a HTTP server for the remote signer API and registering clients
type Server struct {
	ctx                       context.Context
	cancel                    context.CancelFunc
	host                      string // http host, not grpc
	port                      int    // http port, not grpc
	server                    *httprest.Server
	grpcMaxCallRecvMsgSize    int
	grpcRetries               uint
	grpcRetryDelay            time.Duration
	grpcHeaders               []string
	beaconNodeValidatorClient iface.ValidatorClient
	chainClient               iface.ChainClient
	nodeClient                iface.NodeClient
	healthClient              ethpb.HealthClient
	beaconNodeEndpoint        string
	beaconApiEndpoint         string
	beaconApiTimeout          time.Duration
	beaconNodeCert            string
	jwtSecret                 []byte
	authTokenPath             string
	authToken                 string
	db                        db.Database
	walletDir                 string
	wallet                    *wallet.Wallet
	walletInitializedFeed     *event.Feed
	walletInitialized         bool
	validatorService          *client.ValidatorService
	router                    *mux.Router
	logStreamer               logs.Streamer
	logStreamerBufferSize     int
	startFailure              error
}

// NewServer instantiates a new HTTP server.
func NewServer(ctx context.Context, cfg *Config) *Server {
	ctx, cancel := context.WithCancel(ctx)
	server := &Server{
		ctx:                    ctx,
		cancel:                 cancel,
		logStreamer:            logs.NewStreamServer(),
		logStreamerBufferSize:  1000, // Enough to handle most bursts of logs in the validator client.
		host:                   cfg.Host,
		port:                   cfg.Port,
		grpcMaxCallRecvMsgSize: cfg.GRPCMaxCallRecvMsgSize,
		grpcRetries:            cfg.GRPCRetries,
		grpcRetryDelay:         cfg.GRPCRetryDelay,
		grpcHeaders:            cfg.GRPCHeaders,
		validatorService:       cfg.ValidatorService,
		authTokenPath:          cfg.AuthTokenPath,
		db:                     cfg.DB,
		walletDir:              cfg.WalletDir,
		walletInitializedFeed:  cfg.WalletInitializedFeed,
		walletInitialized:      cfg.Wallet != nil,
		wallet:                 cfg.Wallet,
		beaconApiTimeout:       cfg.BeaconApiTimeout,
		beaconApiEndpoint:      cfg.BeaconApiEndpoint,
		beaconNodeEndpoint:     cfg.BeaconNodeGRPCEndpoint,
		router:                 cfg.Router,
	}

	if server.authTokenPath == "" && server.walletDir != "" {
		server.authTokenPath = filepath.Join(server.walletDir, api.AuthTokenFileName)
	}

	if server.authTokenPath != "" {
		if err := server.initializeAuthToken(); err != nil {
			log.WithError(err).Error("Could not initialize web auth token")
		}
		validatorWebAddr := fmt.Sprintf("%s:%d", server.host, server.port)
		logValidatorWebAuth(validatorWebAddr, server.authToken, server.authTokenPath)
		go server.refreshAuthTokenFromFileChanges(server.ctx, server.authTokenPath)
	}
	// Register a gRPC or HTTP client to the beacon node.
	// Used for proxy calls to beacon node from validator REST handlers
	if err := server.registerBeaconClient(); err != nil {
		log.WithError(err).Fatal("Could not register beacon chain gRPC or HTTP client")
	}

	if err := server.InitializeRoutesWithWebHandler(); err != nil {
		log.WithError(err).Fatal("Could not initialize routes with web handler")
	}

	opts := []httprest.Option{
		httprest.WithRouter(cfg.Router),
		httprest.WithHTTPAddr(net.JoinHostPort(server.host, fmt.Sprintf("%d", server.port))),
	}
	// create and set a new http server
	s, err := httprest.New(server.ctx, opts...)
	if err != nil {
		log.WithError(err).Fatal("Failed to create HTTP server")
	}
	server.server = s

	return server
}

// Start the HTTP server and registers clients that can communicate via HTTP or gRPC.
func (s *Server) Start() {
	s.server.Start()
}

// InitializeRoutesWithWebHandler adds a catchall wrapper for web handling
func (s *Server) InitializeRoutesWithWebHandler() error {
	if err := s.InitializeRoutes(); err != nil {
		return err
	}
	s.router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api") {
			r.URL.Path = strings.Replace(r.URL.Path, "/api", "", 1) // used to redirect apis to standard rest APIs
			s.router.ServeHTTP(w, r)
		} else {
			// Finally, we handle with the web server.
			web.Handler(w, r)
		}
	})
	return nil
}

// InitializeRoutes initializes pure HTTP REST endpoints for the validator client.
// needs to be called before the Serve function
func (s *Server) InitializeRoutes() error {
	if s.router == nil {
		return errors.New("no router found on server")
	}
	// Adding Auth Interceptor for the routes below
	s.router.Use(s.AuthTokenHandler)
	// Register all services, HandleFunc calls, etc.
	// ...
	s.router.HandleFunc("/eth/v1/keystores", s.ListKeystores).Methods(http.MethodGet)
	s.router.HandleFunc("/eth/v1/keystores", s.ImportKeystores).Methods(http.MethodPost)
	s.router.HandleFunc("/eth/v1/keystores", s.DeleteKeystores).Methods(http.MethodDelete)
	s.router.HandleFunc("/eth/v1/remotekeys", s.ListRemoteKeys).Methods(http.MethodGet)
	s.router.HandleFunc("/eth/v1/remotekeys", s.ImportRemoteKeys).Methods(http.MethodPost)
	s.router.HandleFunc("/eth/v1/remotekeys", s.DeleteRemoteKeys).Methods(http.MethodDelete)
	s.router.HandleFunc("/eth/v1/validator/{pubkey}/gas_limit", s.GetGasLimit).Methods(http.MethodGet)
	s.router.HandleFunc("/eth/v1/validator/{pubkey}/gas_limit", s.SetGasLimit).Methods(http.MethodPost)
	s.router.HandleFunc("/eth/v1/validator/{pubkey}/gas_limit", s.DeleteGasLimit).Methods(http.MethodDelete)
	s.router.HandleFunc("/eth/v1/validator/{pubkey}/feerecipient", s.ListFeeRecipientByPubkey).Methods(http.MethodGet)
	s.router.HandleFunc("/eth/v1/validator/{pubkey}/feerecipient", s.SetFeeRecipientByPubkey).Methods(http.MethodPost)
	s.router.HandleFunc("/eth/v1/validator/{pubkey}/feerecipient", s.DeleteFeeRecipientByPubkey).Methods(http.MethodDelete)
	s.router.HandleFunc("/eth/v1/validator/{pubkey}/voluntary_exit", s.SetVoluntaryExit).Methods(http.MethodPost)
	s.router.HandleFunc("/eth/v1/validator/{pubkey}/graffiti", s.GetGraffiti).Methods(http.MethodGet)
	s.router.HandleFunc("/eth/v1/validator/{pubkey}/graffiti", s.SetGraffiti).Methods(http.MethodPost)
	s.router.HandleFunc("/eth/v1/validator/{pubkey}/graffiti", s.DeleteGraffiti).Methods(http.MethodDelete)

	// auth endpoint
	s.router.HandleFunc(api.WebUrlPrefix+"initialize", s.Initialize).Methods(http.MethodGet)
	// accounts endpoints
	s.router.HandleFunc(api.WebUrlPrefix+"accounts", s.ListAccounts).Methods(http.MethodGet)
	s.router.HandleFunc(api.WebUrlPrefix+"accounts/backup", s.BackupAccounts).Methods(http.MethodPost)
	s.router.HandleFunc(api.WebUrlPrefix+"accounts/voluntary-exit", s.VoluntaryExit).Methods(http.MethodPost)
	// web health endpoints
	s.router.HandleFunc(api.WebUrlPrefix+"health/version", s.GetVersion).Methods(http.MethodGet)
	s.router.HandleFunc(api.WebUrlPrefix+"health/logs/validator/stream", s.StreamValidatorLogs).Methods(http.MethodGet)
	s.router.HandleFunc(api.WebUrlPrefix+"health/logs/beacon/stream", s.StreamBeaconLogs).Methods(http.MethodGet)
	// Beacon calls
	s.router.HandleFunc(api.WebUrlPrefix+"beacon/status", s.GetBeaconStatus).Methods(http.MethodGet)
	s.router.HandleFunc(api.WebUrlPrefix+"beacon/summary", s.GetValidatorPerformance).Methods(http.MethodGet)
	s.router.HandleFunc(api.WebUrlPrefix+"beacon/validators", s.GetValidators).Methods(http.MethodGet)
	s.router.HandleFunc(api.WebUrlPrefix+"beacon/balances", s.GetValidatorBalances).Methods(http.MethodGet)
	s.router.HandleFunc(api.WebUrlPrefix+"beacon/peers", s.GetPeers).Methods(http.MethodGet)
	// web wallet endpoints
	s.router.HandleFunc(api.WebUrlPrefix+"wallet", s.WalletConfig).Methods(http.MethodGet)
	s.router.HandleFunc(api.WebUrlPrefix+"wallet/create", s.CreateWallet).Methods(http.MethodPost)
	s.router.HandleFunc(api.WebUrlPrefix+"wallet/keystores/validate", s.ValidateKeystores).Methods(http.MethodPost)
	s.router.HandleFunc(api.WebUrlPrefix+"wallet/recover", s.RecoverWallet).Methods(http.MethodPost)
	// slashing protection endpoints
	s.router.HandleFunc(api.WebUrlPrefix+"slashing-protection/export", s.ExportSlashingProtection).Methods(http.MethodGet)
	s.router.HandleFunc(api.WebUrlPrefix+"slashing-protection/import", s.ImportSlashingProtection).Methods(http.MethodPost)

	log.Info("Initialized REST API routes")
	return nil
}

// Stop the HTTP server.
func (s *Server) Stop() error {
	return s.server.Stop()
}

// Status returns an error if the service is unhealthy.
func (s *Server) Status() error {
	if s.startFailure != nil {
		return s.startFailure
	}
	return nil
}
