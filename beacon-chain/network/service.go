package network

import (
	"hash"

	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "network")

// Service is the middleware between the application-agnostic p2p service and subscribers to the network.
type Service struct {
	syncService SyncService
}

// SyncService is the interface for the sync service.
type SyncService interface {
	ReceiveBlockHash(hash.Hash)
	ReceiveBlock(*types.Block) error
}

// NewNetworkService instantiates a new network service.
func NewNetworkService() *Service {
	return &Service{}
}

// SetSyncService sets a concrete value for the sync service.
func (ns *Service) SetSyncService(ss SyncService) {
	ns.syncService = ss
}

// Start launches the service's goroutine.
func (ns *Service) Start() {
	log.Info("Starting service")
	go run()
}

// Stop kills the service's goroutine (unimplemented).
func (ns *Service) Stop() error {
	log.Info("Stopping service")
	return nil
}

// BroadcastBlockHash sends the block hash to other peers in the network.
func (ns *Service) BroadcastBlockHash(h hash.Hash) error {
	return nil
}

// BroadcastBlock sends the block to other peers in the network.
func (ns *Service) BroadcastBlock(b *types.Block) error {
	return nil
}

// RequestBlock requests the contents of the block given the block hash.
func (ns *Service) RequestBlock(hash.Hash) error {
	return nil
}

func run() {
	select {}
}
