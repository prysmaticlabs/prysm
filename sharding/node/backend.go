package node

import (
	"log"

	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/notary"
	"github.com/ethereum/go-ethereum/sharding/observer"
	"github.com/ethereum/go-ethereum/sharding/proposer"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/sharding/params"
	"github.com/ethereum/go-ethereum/sharding/txpool"
	cli "gopkg.in/urfave/cli.v1"
)

// ShardEthereum is a service that is registered and started when geth is launched.
// it contains APIs and fields that handle the different components of the sharded
// Ethereum network.
type ShardEthereum struct {
	shardConfig  *params.ShardConfig    // Holds necessary information to configure shards.
	txPool       *txpool.ShardTxPool    // Defines the sharding-specific txpool. To be designed.
	actor        sharding.ShardingActor // Either notary, proposer, or observer.
	shardChainDb ethdb.Database         // Access to the persistent db to store shard data.
	eventFeed    *event.Feed            // Used to enable P2P related interactions via different sharding actors.
}

// New creates a new sharding-enabled Ethereum service. This is called in the main
// geth sharding entrypoint.
func New(ctx *cli.Context) (*ShardEthereum, error) {

	seth := &ShardEthereum{}

	// path := node.DefaultDataDir()
	// if ctx.GlobalIsSet(utils.DataDirFlag.Name) {
	// 	path = ctx.GlobalString(utils.DataDirFlag.Name)
	// }

	// endpoint := ctx.Args().First()
	// if endpoint == "" {
	// 	endpoint = fmt.Sprintf("%s/%s.ipc", path, clientIdentifier)
	// }
	// if ctx.GlobalIsSet(utils.IPCPathFlag.Name) {
	// 	endpoint = ctx.GlobalString(utils.IPCPathFlag.Name)
	// }

	// config := &node.Config{
	// 	DataDir: path,
	// }

	// scryptN, scryptP, keydir, err := config.AccountConfig()
	// if err != nil {
	// 	return nil, err
	// }

	// ks := keystore.NewKeyStore(keydir, scryptN, scryptP)

	actorFlag := ctx.GlobalString(utils.ActorFlag.Name)

	var actor sharding.ShardingActor

	if actorFlag == "notary" {
		not, err := notary.NewNotary(seth)
		if err != nil {
			return nil, err
		}
		actor = not
	} else if actorFlag == "proposer" {
		prop, err := proposer.NewProposer(seth)
		if err != nil {
			return nil, err
		}
		actor = prop
	} else {
		obs, err := observer.NewObserver(seth)
		if err != nil {
			return nil, err
		}
		actor = obs
	}

	seth.actor = actor
	return nil, nil
}

// Start the ShardEthereum service and kicks off the p2p and actor's main loop.
func (s *ShardEthereum) Start() error {
	log.Println("Starting sharding service")
	if err := s.actor.Start(); err != nil {
		return err
	}
	defer s.actor.Stop()

	// TODO: start p2p and other relevant services.
	return nil
}

// Close handles graceful shutdown of the system.
func (s *ShardEthereum) Close() error {
	// rpcClient could be nil if the connection failed.
	if s.rpcClient != nil {
		s.rpcClient.Close()
	}
	return nil
}
