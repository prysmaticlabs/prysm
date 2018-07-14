package node

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/prysmaticlabs/geth-sharding/beacon-chain/mainchain"
	"github.com/prysmaticlabs/geth-sharding/beacon-chain/types"
	"github.com/prysmaticlabs/geth-sharding/shared"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

// BeaconNode defines a struct that handles the services running a random beacon chain
// full PoS node. It handles the lifecycle of the entire system and registers
// services to a service registry.
type BeaconNode struct {
	ctx      *cli.Context
	services *shared.ServiceRegistry
	lock     sync.RWMutex
	stop     chan struct{} // Channel to wait for termination notifications.
}

// New creates a new node instance, sets up configuration options, and registers
// every required service to the node.
func New(ctx *cli.Context) (*BeaconNode, error) {
	registry := shared.NewServiceRegistry()
	beacon := &BeaconNode{
		ctx:      ctx,
		services: registry,
		stop:     make(chan struct{}),
	}

	if err := beacon.registerWeb3Service(); err != nil {
		return nil, err
	}

	return beacon, nil
}

// Start the BeaconNode and kicks off every registered service.
func (b *BeaconNode) Start() {
	b.lock.Lock()

	log.Info("Starting beacon node")

	b.services.StartAll()

	stop := b.stop
	b.lock.Unlock()

	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sigc)
		<-sigc
		log.Info("Got interrupt, shutting down...")
		go b.Close()
		for i := 10; i > 0; i-- {
			<-sigc
			if i > 1 {
				log.Info("Already shutting down, interrupt more to panic.", "times", i-1)
			}
		}
		// Ensure trace and CPU profile data is flushed.
		panic("Panic closing the beacon node")
	}()

	// Wait for stop channel to be closed.
	<-stop
}

// Close handles graceful shutdown of the system.
func (b *BeaconNode) Close() {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.services.StopAll()
	log.Info("Stopping beacon node")
	close(b.stop)
}

func (b *BeaconNode) registerWeb3Service() error {
	endpoint := b.ctx.GlobalString(types.Web3ProviderFlag.Name)
	web3Service, err := mainchain.NewWeb3Service(endpoint)
	if err != nil {
		return fmt.Errorf("could not register web3Service: %v", err)
	}
	return b.services.RegisterService(web3Service)
}
