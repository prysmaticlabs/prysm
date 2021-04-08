package subscriber

import (
	"context"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/beacon"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/subscriber/api"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/subscriber/api/events"
	"sync"
	"time"
)

// Config
type Config struct {
	HTTPEnable       bool
	WSEnable         bool
	WSPort           int
	HTTPPort         int
	WSHost           string
	WSPathPrefix     string
	HTTPPathPrefix   string
	IPCPath          string
	HTTPHost         string
	WSOrigins        []string
	HTTPVirtualHosts []string
	HTTPModules      []string
	HTTPTimeouts     rpc.HTTPTimeouts
	HTTPCors         []string
}

// Service
type Service struct {
	isRunning      bool
	processingLock sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
	runError       error
	stop           chan struct{} // Channel to wait for termination notifications

	backend       *api.APIBackend
	config        *Config
	rpcAPIs       []rpc.API   // List of APIs currently provided by the node
	http          *httpServer //
	ws            *httpServer //
	ipc           *ipcServer  // Stores information about the ipc http server
	inprocHandler *rpc.Server // In-process RPC request handler to process the API requests
}

func NewService(ctx context.Context, cfg *Config, server beacon.Server) (*Service, error) {
	ctx, cancel := context.WithCancel(ctx)
	_ = cancel // govet fix for lost cancel. Cancel is handled in service.Stop()

	service := &Service{
		ctx:           ctx,
		cancel:        cancel,
		config:        cfg,
		inprocHandler: rpc.NewServer(),
		backend: &api.APIBackend{
			BeaconChain: server,
		},
	}

	// Configure RPC servers.
	service.rpcAPIs = service.APIs()
	service.http = newHTTPServer(rpc.DefaultHTTPTimeouts)
	service.ws = newHTTPServer(rpc.DefaultHTTPTimeouts)
	service.ipc = newIPCServer(service.config.IPCPath)

	return service, nil
}

// Start a consensus info fetcher service's main event loop.
func (s *Service) Start() {
	if s.isRunning {
		log.Error("Attempted to start rpc server when it was already started")
		return
	}

	go func() {
		// start RPC endpoints
		err := s.startRPC()
		if err != nil {
			s.stopRPC()
			s.runError = err
			log.Errorf("Could not serve vanguard RPC: %v", err)
		}
	}()
}

// Stop
func (s *Service) Stop() error {
	if s.cancel != nil {
		defer s.cancel()
	}
	s.stopRPC()
	return nil
}

// Status
func (s *Service) Status() error {
	// Service don't start
	if !s.isRunning {
		return nil
	}
	// get error from run function
	if s.runError != nil {
		return s.runError
	}
	return nil
}

// configureRPC is a helper method to configure all the various RPC endpoints during node
// startup. It's not meant to be called at any time afterwards as it makes certain
// assumptions about the state of the node.
func (s *Service) startRPC() error {
	if err := s.startInProc(); err != nil {
		return err
	}

	// Configure IPC.
	if s.ipc.endpoint != "" {
		log.WithField("ipcEndpoint", s.ipc.endpoint).Info("starting ipc endpoint")
		if err := s.ipc.start(s.rpcAPIs); err != nil {
			return err
		}
	}

	// Configure HTTP.
	if s.config.HTTPEnable && s.config.HTTPHost != "" {
		config := httpConfig{
			CorsAllowedOrigins: nil,
			Vhosts:             nil,
			Modules:            nil,
			prefix:             "",
		}
		if err := s.http.setListenAddr(s.config.HTTPHost, s.config.HTTPPort); err != nil {
			return err
		}
		if err := s.http.enableRPC(s.rpcAPIs, config); err != nil {
			return err
		}
	}

	// Configure WebSocket.
	if s.config.WSEnable && s.config.WSHost != "" {
		server := s.wsServerForPort(s.config.WSPort)
		config := wsConfig{
			Modules: nil,
			Origins: []string{"*"},
			prefix:  "",
		}
		if err := server.setListenAddr(s.config.WSHost, s.config.WSPort); err != nil {
			return err
		}
		if err := server.enableWS(s.rpcAPIs, config); err != nil {
			return err
		}
	}

	if err := s.http.start(); err != nil {
		return err
	}
	if err := s.ws.start(); err != nil {
		return err
	}

	s.isRunning = true
	return nil
}

// startInProc registers all RPC APIs on the inproc server.
func (s *Service) startInProc() error {
	for _, api := range s.rpcAPIs {
		if err := s.inprocHandler.RegisterName(api.Namespace, api.Service); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) wsServerForPort(port int) *httpServer {
	if s.config.HTTPHost == "" || s.http.port == port {
		return s.http
	}
	return s.ws
}

func (s *Service) stopRPC() {
	s.http.stop()
	s.ws.stop()
	err := s.ipc.stop()
	if nil != err {
		log.Errorf("stopping RPC err = %s", err.Error())
	}
	s.stopInProc()
}

// stopInProc terminates the in-process RPC endpoint.
func (s *Service) stopInProc() {
	s.inprocHandler.Stop()
}

// Wait blocks until the node is closed.
func (s *Service) Wait() {
	<-s.stop
}

func (s *Service) APIs() []rpc.API {
	// Append all the local APIs and return
	return []rpc.API{
		{
			Namespace: "van",
			Version:   "1.0",
			Service:   events.NewPublicFilterAPI(s.backend, 5*time.Minute),
			Public:    true,
		},
	}
}
