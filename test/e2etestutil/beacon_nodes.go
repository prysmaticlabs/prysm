package e2etestutil

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/node"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/urfave/cli"
)

type BeaconNodesInstance struct {
	nodes []*node.BeaconNode
	t     *testing.T
	geth  *GoEthereumInstance
}

func NewBeaconNodes(t *testing.T, instances int, geth *GoEthereumInstance) *BeaconNodesInstance {
	// Clear datadirs
	if err := os.RemoveAll(testutil.TempDir() + "/beacon"); err != nil {
		t.Fatal(err)
	}

	var nodes []*node.BeaconNode
	for i := 0; i < instances; i++ {
		flagSet := flag.NewFlagSet("test", 0)
		flagSet.String(utils.DepositContractFlag.Name, geth.DepositContractAddr.String(), "")
		flagSet.String(utils.Web3ProviderFlag.Name, "ws://127.0.0.1:9000", "")
		flagSet.String(cmd.DataDirFlag.Name, fmt.Sprintf("%s/beacon/db%d", testutil.TempDir(), i), "")
		flagSet.Uint64(utils.ChainStartDelay.Name, chainStartDelay.Uint64(), "")
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
		go n.Start()
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
