package simulator

import (
	"context"
	"crypto/rand"
	"math/big"
	mrand "math/rand"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/prysmaticlabs/prysm/client/mainchain"
	"github.com/prysmaticlabs/prysm/client/params"
	"github.com/prysmaticlabs/prysm/client/syncer"
	"github.com/prysmaticlabs/prysm/shared/p2p"

	pb "github.com/prysmaticlabs/prysm/proto/sharding/v1"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "simulator")

// Simulator is a service in a shard node that simulates requests from
// remote notes coming over the shardp2p network. For example, if
// we are running a proposer service, we would want to simulate notary requests
// requests coming to us via a p2p feed. This service will be removed
// once p2p internals and end-to-end testing across remote
// nodes have been implemented.
type Simulator struct {
	config      *params.Config
	client      *mainchain.SMCClient
	p2p         *p2p.Server
	shardID     int
	ctx         context.Context
	cancel      context.CancelFunc
	delay       time.Duration
	requestFeed *event.Feed
}

// NewSimulator creates a struct instance of a simulator service.
// It will have access to config, a mainchain client, a p2p server,
// and a shardID.
func NewSimulator(config *params.Config, client *mainchain.SMCClient, p2p *p2p.Server, shardID int, delay time.Duration) (*Simulator, error) {
	ctx, cancel := context.WithCancel(context.Background())
	return &Simulator{config, client, p2p, shardID, ctx, cancel, delay, nil}, nil
}

// Start the main loop for simulating p2p requests.
func (s *Simulator) Start() {
	log.Info("Starting simulator service")
	s.requestFeed = s.p2p.Feed(pb.CollationBodyRequest{})

	go s.broadcastTransactions(time.NewTicker(s.delay).C, s.ctx.Done())
	go s.simulateNotaryRequests(s.client.SMCCaller(), s.client.ChainReader(), time.NewTicker(s.delay).C, s.ctx.Done())
}

// Stop the main loop for simulator requests.
func (s *Simulator) Stop() error {
	// Triggers a cancel call in the service's context which shuts down every goroutine
	// in this service.
	defer s.cancel()
	log.Info("Stopping simulator service")
	return nil
}

// simulateNotaryRequests simulates
// requests for collation bodies that will be relayed to the appropriate proposer
// by the p2p feed layer.
func (s *Simulator) simulateNotaryRequests(fetcher mainchain.RecordFetcher, reader mainchain.Reader, delayChan <-chan time.Time, done <-chan struct{}) {
	for {
		select {
		// Makes sure to close this goroutine when the service stops.
		case <-done:
			log.Debug("Simulator context closed, exiting goroutine")
			return
		case <-delayChan:
			blockNumber, err := reader.BlockByNumber(s.ctx, nil)
			if err != nil {
				log.Errorf("Could not fetch current block number: %v", err)
				continue
			}

			period := new(big.Int).Div(blockNumber.Number(), big.NewInt(s.config.PeriodLength))
			// Collation for current period may not exist yet, so let's ask for
			// the collation at period - 1.
			if period.Cmp(big.NewInt(0)) > 0 {
				period = period.Sub(period, big.NewInt(1))
			}
			req, err := syncer.RequestCollationBody(fetcher, big.NewInt(int64(s.shardID)), period)
			if err != nil {
				log.Errorf("Error constructing collation body request: %v", err)
				continue
			}

			if req != nil {
				s.p2p.Broadcast(req)
				log.Debug("Sent request for collation body via a shardp2p broadcast")
			} else {
				log.Warn("Syncer generated nil CollationBodyRequest")
			}
		}
	}
}

// broadcastTransactions sends a transaction with random bytes over by a delay period,
// this method is for testing purposes only, and will be replaced by a more functional CLI tool.
func (s *Simulator) broadcastTransactions(delayChan <-chan time.Time, done <-chan struct{}) {
	for {
		select {
		// Makes sure to close this goroutine when the service stops.
		case <-done:
			log.Debug("Simulator context closed, exiting goroutine")
			return
		case <-delayChan:
			tx := createTestTx()
			s.p2p.Broadcast(tx)
			log.Debug("Transaction broadcasted")
		}
	}
}

// createTestTx is a helper method to generate tx with random data bytes.
// it is used for broadcastTransactions.
func createTestTx() *pb.Transaction {
	data := make([]byte, 1024)
	rand.Read(data)
	// TODO: add more fields.
	return &pb.Transaction{
		Nonce: mrand.Uint64(),
		Input: data,
	}
}
