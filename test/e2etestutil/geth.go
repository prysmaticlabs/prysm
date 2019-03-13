package e2etestutil

import (
	"testing"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
)

var blockPeriod = uint64(0) // seconds

type GoEthereumInstance struct {
	DepositContractAddr common.Address

	t               *testing.T
	node            *node.Node
	ks              *keystore.KeyStore
	depositContract *contracts.DepositContract
	client          *ethclient.Client
}

func NewGoEthereumInstance(t *testing.T) *GoEthereumInstance {
	cfg := &node.Config{
		Name: "geth",
		P2P: p2p.Config{
			MaxPeers:    0,
			ListenAddr:  ":0",
			NoDiscovery: true,
			DiscoveryV5: false,
		},
		NoUSB:       true,
		DataDir:     "", // Use memory db
		WSHost:      "127.0.0.1",
		WSPort:      9000,
		WSExposeAll: true,
	}
	node, err := node.New(cfg)
	if err != nil {
		panic(err)
	}

	keystores := node.AccountManager().Backends(keystore.KeyStoreType)
	ks := keystores[0].(*keystore.KeyStore)

	devAcct, err := ks.NewAccount("")
	if err != nil {
		t.Fatal(err)
	}
	if err := ks.Unlock(devAcct, ""); err != nil {
		t.Fatal(err)
	}

	ethCfg := &eth.Config{
		Genesis:   core.DeveloperGenesisBlock(blockPeriod, devAcct.Address),
		NetworkId: 1337,
		Etherbase: devAcct.Address,
	}
	utils.RegisterEthService(node, ethCfg)

	return &GoEthereumInstance{
		t:    t,
		node: node,
		ks:   ks,
	}
}

func (g *GoEthereumInstance) Start() {
	if err := g.node.Start(); err != nil {
		g.t.Fatal(err)
	}
	var ethereum *eth.Ethereum
	if err := g.node.Service(&ethereum); err != nil {
		g.t.Fatalf("Ethereum service not running: %v", err)
	}
	if err := ethereum.StartMining(1 /*threads*/); err != nil {
		g.t.Fatal(err)
	}

	client, err := g.node.Attach()
	if err != nil {
		g.t.Fatal(err)
	}
	g.client = ethclient.NewClient(client)
}

func (g *GoEthereumInstance) Stop() error {
	return g.node.Stop()
}

func (g *GoEthereumInstance) Status() error {
	return nil
}
