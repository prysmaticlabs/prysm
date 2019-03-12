package e2etestutil

import (
	"flag"
	"fmt"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/validator/node"
	"github.com/prysmaticlabs/prysm/validator/types"
	"github.com/urfave/cli"
)

type ValidatorsInstance struct {
	clients []*node.ValidatorClient
	t       *testing.T
}

func NewValidators(t *testing.T, numValidators int, beacons *BeaconNodesInstance) *ValidatorsInstance {

	numBeaconNodes := len(beacons.NodeGRPCAddrs)
	var clients []*node.ValidatorClient
	for i := 0; i < numValidators; i++ {
		GRPCAddr := beacons.NodeGRPCAddrs[i%numBeaconNodes]
		keystorePath := fmt.Sprintf("%s/keystore%d", testutil.TempDir(), i)

		flagSet := flag.NewFlagSet("test", 0)
		flagSet.String(types.BeaconRPCProviderFlag.Name, GRPCAddr, "")
		flagSet.String(types.KeystorePathFlag.Name, keystorePath, "")
		flagSet.String(types.PasswordFlag.Name, "", "")
		v, err := node.NewValidatorClient(cli.NewContext(
			cli.NewApp(),
			flagSet,
			nil, /* parentContext */
		))
		if err != nil {
			t.Fatal(err)
		}

		clients = append(clients, v)
	}

	return &ValidatorsInstance{
		clients: clients,
		t:       t,
	}
}

func (v *ValidatorsInstance) Start() {
	for _, client := range v.clients {
		go client.Start()
	}
}

func (v *ValidatorsInstance) Stop() error {
	for _, client := range v.clients {
		client.Close()
	}
	return nil
}

func (v *ValidatorsInstance) Status() error {
	return nil
}
