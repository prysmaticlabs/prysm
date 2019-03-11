package e2etestutil

import (
	"flag"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/node"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	"github.com/urfave/cli"
)

type BeaconNodesInstance struct {
	nodes []*node.BeaconNode
	t     *testing.T
	geth  *GoEthereumInstance
}

func NewBeaconNodes(t *testing.T, instances int, geth *GoEthereumInstance) *BeaconNodesInstance {
	var nodes []*node.BeaconNode
	flagSet := flag.NewFlagSet("test", 0)
	flagSet.String(utils.DepositContractFlag.Name, geth.DepositContractAddr.String(), "")
	for i := 0; i < instances; i++ {
		n, err := node.NewBeaconNode(cli.NewContext(
			cli.NewApp(),
			flagSet,
			nil, /* parentContext */
		))
		if err != nil {
			t.Fatal(err)
		}
		nodes = append(nodes, n)
	}
	return &BeaconNodesInstance{
		nodes: nodes,
		t:     t,
		geth:  geth,
	}

}

func (b *BeaconNodesInstance) Start() {
	for _, n := range b.nodes {
		n.Start()
	}
}

func (b *BeaconNodesInstance) Stop() error {
	for _, n := range b.nodes {
		n.Close()
	}
	return nil
}

func (b *BeaconNodesInstance) Status() error {
	return nil
}
