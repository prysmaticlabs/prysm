package mainchain

import (
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
)

// Web3Service fetches important information about the canonical
// Ethereum PoW chain via a web3 endpoint using an ethclient. The Random
// Beacon Chain requires synchronization with the mainchain's current
// blockhash, block number, and access to logs within the
// Validator Registration Contract on the mainchain to kick off the beacon
// chain's validator registration process.
type Web3Service struct {
	client *ethclient.Client
}

// NewWeb3Service sets up a new instance with an ethclient when
// given a web3 endpoint as a string.
func NewWeb3Service(endpoint string) (*Web3Service, error) {
	return &Web3Service{}, nil
}

// Start a web3 service's main event loop.
func (w *Web3Service) Start() {
	log.Info("Starting web3 mainchain service")
}

// Stop the web3 service's main event loop and associated goroutines.
func (w *Web3Service) Stop() error {
	log.Info("Stopping web3 mainchain service")
	return nil
}
