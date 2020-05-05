package node

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"

	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"gopkg.in/urfave/cli.v2"
)

// Ensure BeaconNode implements interfaces.
var _ = statefeed.Notifier(&BeaconNode{})

// Test that beacon chain node can close.
func TestNodeClose_OK(t *testing.T) {
	hook := logTest.NewGlobal()

	tmp := fmt.Sprintf("%s/datadirtest2", testutil.TempDir())
	if err := os.RemoveAll(tmp); err != nil {
		t.Log(err)
	}

	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String("web3provider", "ws//127.0.0.1:8546", "web3 provider ws or IPC endpoint")
	set.Bool("test-skip-pow", true, "skip pow dial")
	set.String("datadir", tmp, "node data directory")
	set.String("p2p-encoding", "ssz", "p2p encoding scheme")
	set.Bool("demo-config", true, "demo configuration")
	set.String("deposit-contract", "0x0000000000000000000000000000000000000000", "deposit contract address")

	context := cli.NewContext(&app, set, nil)

	node, err := NewBeaconNode(context)
	if err != nil {
		t.Fatalf("Failed to create BeaconNode: %v", err)
	}

	node.Close()

	testutil.AssertLogsContain(t, hook, "Stopping beacon node")

	if err := os.RemoveAll(tmp); err != nil {
		t.Log(err)
	}
}

func TestLoadConfigFile(t *testing.T) {
	mainnetConfigFile := testutil.ConfigFilePath(t, "mainnet")
	loadChainConfigFile(mainnetConfigFile)
	if params.BeaconConfig().MaxCommitteesPerSlot != params.MainnetConfig().MaxCommitteesPerSlot {
		t.Errorf("Expected MaxCommitteesPerSlot to be set to mainnet value: %d found: %d",
			params.MainnetConfig().MaxCommitteesPerSlot,
			params.BeaconConfig().MaxCommitteesPerSlot)
	}
	if params.BeaconConfig().SecondsPerSlot != params.MainnetConfig().SecondsPerSlot {
		t.Errorf("Expected SecondsPerSlot to be set to mainnet value: %d found: %d",
			params.MainnetConfig().SecondsPerSlot,
			params.BeaconConfig().SecondsPerSlot)
	}
	minimalConfigFile := testutil.ConfigFilePath(t, "minimal")
	loadChainConfigFile(minimalConfigFile)
	if params.BeaconConfig().MaxCommitteesPerSlot != params.MinimalSpecConfig().MaxCommitteesPerSlot {
		t.Errorf("Expected MaxCommitteesPerSlot to be set to minimal value: %d found: %d",
			params.MinimalSpecConfig().MaxCommitteesPerSlot,
			params.BeaconConfig().MaxCommitteesPerSlot)
	}
	if params.BeaconConfig().SecondsPerSlot != params.MinimalSpecConfig().SecondsPerSlot {
		t.Errorf("Expected SecondsPerSlot to be set to minimal value: %d found: %d",
			params.MinimalSpecConfig().SecondsPerSlot,
			params.BeaconConfig().SecondsPerSlot)
	}
}

func Test_replaceHexStringWithYAMLFormat(t *testing.T) {

	testLines := []struct {
		line   string
		wanted string
	}{
		{
			line:   "FOUR_BYTES: 0x41414141",
			wanted: "FOUR_BYTES: \n- 65\n- 65\n- 65\n- 65\n",
		},
		{
			line:   "THREE_BYTES: 0x414141",
			wanted: "THREE_BYTES: \n- 65\n- 65\n- 65\n- 0\n",
		},
		{
			line:   "EIGHT_BYTES: 0x4141414141414141",
			wanted: "EIGHT_BYTES: \n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n",
		},
		{
			line: "SIXTEEN_BYTES: 0x41414141414141414141414141414141",
			wanted: "SIXTEEN_BYTES: \n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n" +
				"- 65\n- 65\n- 65\n- 65\n",
		},
		{
			line: "TWENTY_BYTES: 0x4141414141414141414141414141414141414141",
			wanted: "TWENTY_BYTES: \n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n" +
				"- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n",
		},
		{
			line: "THIRTY_TWO_BYTES: 0x4141414141414141414141414141414141414141414141414141414141414141",
			wanted: "THIRTY_TWO_BYTES: \n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n" +
				"- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n" +
				"- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n",
		},
		{
			line: "FORTY_EIGHT_BYTES: 0x41414141414141414141414141414141414141414141414141414141414141414141" +
				"4141414141414141414141414141",
			wanted: "FORTY_EIGHT_BYTES: \n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n" +
				"- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n" +
				"- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n" +
				"- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n",
		},
		{
			line: "NINETY_SIX_BYTES: 0x414141414141414141414141414141414141414141414141414141414141414141414141" +
				"4141414141414141414141414141414141414141414141414141414141414141414141414141414141414141414141" +
				"41414141414141414141414141",
			wanted: "NINETY_SIX_BYTES: \n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n" +
				"- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n" +
				"- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n" +
				"- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n" +
				"- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n" +
				"- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n" +
				"- 65\n- 65\n- 65\n- 65\n- 65\n- 65\n",
		},
	}
	for _, line := range testLines {
		parts := replaceHexStringWithYAMLFormat(line.line)
		res := strings.Join(parts, "\n")

		if res != line.wanted {
			t.Errorf("expected conversion to be: %v got: %v", line.wanted, res)
		}
	}
}
