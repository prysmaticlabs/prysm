package endtoend

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

var slasherLogFileName = "slasher-%d.log"

// startSlasher starts a slasher client for use within E2E, connected to the first beacon node.
// It returns the process ID of the slasher.
func startSlashers(t *testing.T, config *end2EndConfig) []int {
	tmpPath := config.tmpPath
	binaryPath, found := bazel.FindBinary("slasher", "slasher")
	if !found {
		t.Log(binaryPath)
		t.Fatal("Slasher binary not found")
	}

	var processIDs []int
	for i := uint64(0); i < config.numBeaconNodes; i++ {
		stdOutFile, err := deleteAndCreateFile(tmpPath, fmt.Sprintf(slasherLogFileName, i))
		if err != nil {
			t.Fatal(err)
		}

		args := []string{
			fmt.Sprintf("--datadir=%s/slasher-data-%d/", tmpPath, i),
			fmt.Sprintf("--log-file=%s", stdOutFile.Name()),
			"--force-clear-db",
			"--span-map-cache",
			"--verbosity=debug",
			fmt.Sprintf("--beacon-rpc-provider=localhost:%d", 4200+i),
		}

		t.Logf("Starting slasher %d with flags: %s", i, strings.Join(args[2:], " "))
		cmd := exec.Command(binaryPath, args...)
		if err := cmd.Start(); err != nil {
			t.Fatalf("Failed to start slasher client: %v", err)
		}
		processIDs = append(processIDs, cmd.Process.Pid)
	}

	stdOutFile, err := os.Open(path.Join(tmpPath, fmt.Sprintf(slasherLogFileName, 0)))
	if err != nil {
		t.Fatal(err)
	}
	if err = waitForTextInFile(stdOutFile, "Beacon node is fully synced, starting slashing detection"); err != nil {
		t.Fatalf("could not find starting logs for slasher, this means it had issues starting: %v", err)
	}

	return processIDs
}
