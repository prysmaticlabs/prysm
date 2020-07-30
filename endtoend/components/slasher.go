package components

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/prysmaticlabs/prysm/endtoend/helpers"
	e2e "github.com/prysmaticlabs/prysm/endtoend/params"
)

// StartSlashers starts slasher clients for use within E2E, connected to all beacon nodes.
// It returns the process IDs of the slashers.
func StartSlashers(t *testing.T) {
	for i := 0; i < e2e.TestParams.BeaconNodeCount; i++ {
		startSlasher(t, i)
	}

	stdOutFile, err := os.Open(path.Join(e2e.TestParams.LogPath, fmt.Sprintf(e2e.SlasherLogFileName, 0)))
	if err != nil {
		t.Fatal(err)
	}
	if err = helpers.WaitForTextInFile(stdOutFile, "Beacon node is fully synced, starting slashing detection"); err != nil {
		t.Fatalf("could not find starting logs for slasher, this means it had issues starting: %v", err)
	}
}

func startSlasher(t *testing.T, i int) {
	binaryPath, found := bazel.FindBinary("slasher", "slasher")
	if !found {
		t.Log(binaryPath)
		t.Fatal("Slasher binary not found")
	}

	stdOutFile, err := helpers.DeleteAndCreateFile(e2e.TestParams.LogPath, fmt.Sprintf(e2e.SlasherLogFileName, i))
	if err != nil {
		t.Fatal(err)
	}

	args := []string{
		fmt.Sprintf("--datadir=%s/slasher-data-%d/", e2e.TestParams.TestPath, i),
		fmt.Sprintf("--log-file=%s", stdOutFile.Name()),
		fmt.Sprintf("--rpc-port=%d", e2e.TestParams.SlasherRPCPort+i),
		fmt.Sprintf("--monitoring-port=%d", e2e.TestParams.SlasherMetricsPort+i),
		fmt.Sprintf("--beacon-rpc-provider=localhost:%d", e2e.TestParams.BeaconNodeRPCPort+i),
		"--force-clear-db",
		"--e2e-config",
	}

	t.Logf("Starting slasher %d with flags: %s", i, strings.Join(args[2:], " "))
	cmd := exec.Command(binaryPath, args...)
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start slasher client: %v", err)
	}
}
