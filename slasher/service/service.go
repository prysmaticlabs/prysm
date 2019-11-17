// Package service defines the service used to retrieve slashings proofs and
// feed attestations and block headers into the slasher db.
package service

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/debug"
	"github.com/prysmaticlabs/prysm/shared/version"
	"github.com/prysmaticlabs/prysm/slasher/rpc"
	"github.com/urfave/cli"
	"go.opencensus.io/plugin/ocgrpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"

	middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/slasher/db"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var log logrus.FieldLogger

const slasherDBName = "slasherdata"

func init() {
	log = logrus.WithField("prefix", "slasherRPC")
}

// Service defining an RPC server for the slasher service.
type Service struct {
	slasherDb       *db.Store
	grpcServer      *grpc.Server
	port            string
	withCert        string
	withKey         string
	listener        net.Listener
	credentialError error
	failStatus      error
	ctx             *cli.Context
	lock            sync.RWMutex
	stop            chan struct{} // Channel to wait for termination notifications.
}

// Config options for the slasher server.
type Config struct {
	Port      string
	CertFlag  string
	KeyFlag   string
	SlasherDb *db.Store
}

// NewRPCService creates a new instance of a struct implementing the SlasherService
// interface.
func NewRPCService(cfg *Config, ctx *cli.Context) (*Service, error) {
	s := &Service{
		slasherDb: cfg.SlasherDb,
		port:      cfg.Port,
		withCert:  cfg.CertFlag,
		withKey:   cfg.KeyFlag,
		ctx:       ctx,
		stop:      make(chan struct{}),
	}
	if err := s.startDB(s.ctx); err != nil {
		return nil, err
	}

	return s, nil
}

// Start the gRPC server.
func (s *Service) Start() {
	s.lock.Lock()
	log.WithFields(logrus.Fields{
		"version": version.GetVersion(),
	}).Info("Starting hash slinging slasher node")
	s.startSlasher()
	stop := s.stop
	s.lock.Unlock()

	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sigc)
		<-sigc
		log.Info("Got interrupt, shutting down...")
		debug.Exit(s.ctx) // Ensure trace and CPU profile data are flushed.
		go s.Close()
		for i := 10; i > 0; i-- {
			<-sigc
			if i > 1 {
				log.Info("Already shutting down, interrupt more to panic", "times", i-1)
			}
		}
		panic("Panic closing the hash slinging slasher node")
	}()

	// Wait for stop channel to be closed.
	select {
	case <-stop:
		return
	default:
	}

}
func (s *Service) startSlasher() {
	log.Info("Starting service on port: ", s.port)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", s.port))
	if err != nil {
		log.Errorf("Could not listen to port in Start() :%s: %v", s.port, err)
	}
	s.listener = lis
	log.WithField("port", s.port).Info("Listening on port")

	opts := []grpc.ServerOption{
		grpc.StatsHandler(&ocgrpc.ServerHandler{}),
		grpc.StreamInterceptor(middleware.ChainStreamServer(
			recovery.StreamServerInterceptor(),
			grpc_prometheus.StreamServerInterceptor,
		)),
		grpc.UnaryInterceptor(middleware.ChainUnaryServer(
			recovery.UnaryServerInterceptor(),
			grpc_prometheus.UnaryServerInterceptor,
		)),
	}
	// TODO(#791): Utilize a certificate for secure connections
	// between beacon nodes and validator clients.
	if s.withCert != "" && s.withKey != "" {
		creds, err := credentials.NewServerTLSFromFile(s.withCert, s.withKey)
		if err != nil {
			log.Errorf("Could not load TLS keys: %s", err)
			s.credentialError = err
		}
		opts = append(opts, grpc.Creds(creds))
	} else {
		log.Warn("You are using an insecure gRPC connection! Provide a certificate and key to connect securely")
	}
	s.grpcServer = grpc.NewServer(opts...)
	slasherServer := rpc.Server{
		SlasherDB: s.slasherDb,
	}

	ethpb.RegisterSlasherServer(s.grpcServer, &slasherServer)

	// Register reflection service on gRPC server.
	reflection.Register(s.grpcServer)

	go func() {
		if s.listener != nil {
			if err := s.grpcServer.Serve(s.listener); err != nil {
				log.Errorf("Could not serve gRPC: %v", err)
			}
		}
	}()
}

// Stop the service.
func (s *Service) Stop() error {
	log.Info("Stopping service")
	if s.slasherDb != nil {
		s.slasherDb.Close()
	}
	if s.listener != nil {
		s.grpcServer.GracefulStop()
		log.Debug("Initiated graceful stop of gRPC server")
	}
	return nil
}

// Close handles graceful shutdown of the system.
func (s *Service) Close() {
	s.lock.Lock()
	defer s.lock.Unlock()

	log.Info("Stopping hash slinging slasher")
	s.Stop()
	if err := s.slasherDb.Close(); err != nil {
		log.Errorf("Failed to close slasher database: %v", err)
	}
	close(s.stop)
}

// Status returns nil, credentialError or fail status.
func (s *Service) Status() error {
	if s.credentialError != nil {
		return s.credentialError
	}
	if s.failStatus != nil {
		return s.failStatus
	}
	return nil
}

func (s *Service) startDB(ctx *cli.Context) error {
	baseDir := ctx.GlobalString(cmd.DataDirFlag.Name)
	dbPath := path.Join(baseDir, slasherDBName)
	d, err := db.NewDB(dbPath)
	if err != nil {
		return err
	}
	if s.ctx.GlobalBool(cmd.ClearDB.Name) {
		if err := d.ClearDB(); err != nil {
			return err
		}
		d, err = db.NewDB(dbPath)
		if err != nil {
			return err
		}
	}

	log.WithField("path", dbPath).Info("Checking db")
	s.slasherDb = d
	return nil
}
