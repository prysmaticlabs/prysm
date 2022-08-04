package node

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/builder"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	mockPOW "github.com/prysmaticlabs/prysm/beacon-chain/powchain/testing"
	"github.com/prysmaticlabs/prysm/cmd"
	"github.com/prysmaticlabs/prysm/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/config/features"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/runtime"
	"github.com/prysmaticlabs/prysm/runtime/interop"
	"github.com/prysmaticlabs/prysm/testing/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"github.com/urfave/cli/v2"
)

// Ensure BeaconNode implements interfaces.
var _ statefeed.Notifier = (*BeaconNode)(nil)

// Test that beacon chain node can close.
func TestNodeClose_OK(t *testing.T) {
	hook := logTest.NewGlobal()

	tmp := fmt.Sprintf("%s/datadirtest2", t.TempDir())

	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.Bool("test-skip-pow", true, "skip pow dial")
	set.String("datadir", tmp, "node data directory")
	set.String("p2p-encoding", "ssz", "p2p encoding scheme")
	set.Bool("demo-config", true, "demo configuration")
	set.String("deposit-contract", "0x0000000000000000000000000000000000000000", "deposit contract address")
	set.Bool(cmd.EnableBackupWebhookFlag.Name, true, "")
	require.NoError(t, set.Set(cmd.EnableBackupWebhookFlag.Name, "true"))
	set.String(cmd.BackupWebhookOutputDir.Name, "datadir", "")
	context := cli.NewContext(&app, set, nil)

	node, err := New(context)
	require.NoError(t, err)

	node.Close()

	require.LogsContain(t, hook, "Stopping beacon node")
}

func TestNodeStart_Ok(t *testing.T) {
	hook := logTest.NewGlobal()
	app := cli.App{}
	tmp := fmt.Sprintf("%s/datadirtest2", t.TempDir())
	set := flag.NewFlagSet("test", 0)
	set.String("datadir", tmp, "node data directory")

	ctx := cli.NewContext(&app, set, nil)
	node, err := New(ctx, WithBlockchainFlagOptions([]blockchain.Option{}),
		WithBuilderFlagOptions([]builder.Option{}),
		WithPowchainFlagOptions([]powchain.Option{}))
	require.NoError(t, err)
	node.services = &runtime.ServiceRegistry{}
	go func() {
		node.Start()
	}()
	time.Sleep(3 * time.Second)
	node.Close()
	require.LogsContain(t, hook, "Starting beacon node")

}

func TestNodeStart_Ok_registerDeterminsticGenesisService(t *testing.T) {
	numValidators := uint64(1)
	hook := logTest.NewGlobal()
	app := cli.App{}
	tmp := fmt.Sprintf("%s/datadirtest2", t.TempDir())
	set := flag.NewFlagSet("test", 0)
	set.String("datadir", tmp, "node data directory")
	set.Uint64(flags.InteropNumValidatorsFlag.Name, numValidators, "")
	genesisState, _, err := interop.GenerateGenesisState(context.Background(), 0, numValidators)
	require.NoError(t, err, "Could not generate genesis beacon state")
	for i := uint64(1); i < 2; i++ {
		someRoot := [32]byte{}
		someKey := [fieldparams.BLSPubkeyLength]byte{}
		copy(someRoot[:], strconv.Itoa(int(i)))
		copy(someKey[:], strconv.Itoa(int(i)))
		genesisState.Validators = append(genesisState.Validators, &ethpb.Validator{
			PublicKey:                  someKey[:],
			WithdrawalCredentials:      someRoot[:],
			EffectiveBalance:           params.BeaconConfig().MaxEffectiveBalance,
			Slashed:                    false,
			ActivationEligibilityEpoch: 1,
			ActivationEpoch:            1,
			ExitEpoch:                  1,
			WithdrawableEpoch:          1,
		})
		genesisState.Balances = append(genesisState.Balances, params.BeaconConfig().MaxEffectiveBalance)
	}
	genesisBytes, err := genesisState.MarshalSSZ()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile("genesis_ssz.json", genesisBytes, 0666))
	set.String(flags.InteropGenesisStateFlag.Name, "genesis_ssz.json", "")
	ctx := cli.NewContext(&app, set, nil)
	node, err := New(ctx, WithBlockchainFlagOptions([]blockchain.Option{}),
		WithBuilderFlagOptions([]builder.Option{}),
		WithPowchainFlagOptions([]powchain.Option{}))
	require.NoError(t, err)
	node.services = &runtime.ServiceRegistry{}
	go func() {
		node.Start()
	}()
	time.Sleep(3 * time.Second)
	node.Close()
	require.LogsContain(t, hook, "Starting beacon node")
	require.NoError(t, os.Remove("genesis_ssz.json"))
}

func TestGenerateGenesisSSZ(t *testing.T) {
	data, err := os.ReadFile("../../testing/endtoend/static-files/eth1/genesis.json")
	require.NoError(t, err)
	genesisState := &ethpb.BeaconState{}
	err = json.Unmarshal(data, genesisState)
	require.NoError(t, err)
	bytes, err := genesisState.MarshalSSZ()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile("genesis_ssz.json", bytes, 0666))
}

// TestClearDB tests clearing the database
func TestClearDB(t *testing.T) {
	hook := logTest.NewGlobal()
	srv, endpoint, err := mockPOW.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		srv.Stop()
	})

	tmp := filepath.Join(t.TempDir(), "datadirtest")

	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String("datadir", tmp, "node data directory")
	set.Bool(cmd.ForceClearDB.Name, true, "force clear db")

	context := cli.NewContext(&app, set, nil)
	_, err = New(context, WithPowchainFlagOptions([]powchain.Option{
		powchain.WithHttpEndpoints([]string{endpoint}),
	}))
	require.NoError(t, err)

	require.LogsContain(t, hook, "Removing database")
}

func TestStartSlasherDB_ForceClearDb(t *testing.T) {
	hook := logTest.NewGlobal()
	features.Init(&features.Flags{EnableSlasher: true})

	tmp := filepath.Join(t.TempDir(), "datadirtest")

	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String("datadir", tmp, "node data directory")
	set.Bool(cmd.ClearDB.Name, true, "clear db")
	set.Bool(cmd.ForceClearDB.Name, true, "force clear db")
	set.Bool("slasher", true, "enable slasher")

	ctx := cli.NewContext(&app, set, nil)
	_, err := New(ctx, WithBlockchainFlagOptions([]blockchain.Option{}))
	require.NoError(t, err)

	require.LogsContain(t, hook, "Removing database")
}
