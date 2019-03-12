package e2etestutil

import (
	"flag"
	"fmt"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/validator/accounts"
	"github.com/prysmaticlabs/prysm/validator/node"
	"github.com/prysmaticlabs/prysm/validator/types"
	"github.com/urfave/cli"
)

type ValidatorsInstance struct {
	DepositData [][]byte
	clients     []*node.ValidatorClient
	t           *testing.T
}

func NewValidators(t *testing.T, numValidators int, beacons *BeaconNodesInstance) *ValidatorsInstance {

	numBeaconNodes := len(beacons.NodeGRPCAddrs)
	var clients []*node.ValidatorClient
	var depositData [][]byte
	for i := 0; i < numValidators; i++ {
		GRPCAddr := beacons.NodeGRPCAddrs[i%numBeaconNodes]
		keystorePath := fmt.Sprintf("%s/keystore%d", testutil.TempDir(), i)
		depositDatum, err := accounts.NewValidatorAccount(keystorePath, "password")
		if err != nil {
			t.Fatal(err)
		}

		flagSet := flag.NewFlagSet("test", 0)
		flagSet.String(types.BeaconRPCProviderFlag.Name, GRPCAddr, "")
		flagSet.String(types.KeystorePathFlag.Name, keystorePath, "")
		flagSet.String(types.PasswordFlag.Name, "password", "")
		v, err := node.NewValidatorClient(cli.NewContext(
			cli.NewApp(),
			flagSet,
			nil, /* parentContext */
		))
		if err != nil {
			t.Fatal(err)
		}

		clients = append(clients, v)
		depositData = append(depositData, depositDatum)
	}

	return &ValidatorsInstance{
		DepositData: depositData,
		clients:     clients,
		t:           t,
	}
}

func (v *ValidatorsInstance) Start() {
	for _, client := range v.clients {
		go client.Start()
	}
}

func (v *ValidatorsInstance) Stop() error {
	for _, client := range v.clients {
		//client.Close()
		_ = client
	}
	return nil
}

func (v *ValidatorsInstance) Status() error {
	return nil
}
