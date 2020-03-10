package endtoend

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

var slasherLogFileName = "slasher.log"

// startSlasher starts a slasher client for use within E2E, connected to the first beacon node.
// It returns the process ID of the slasher.
func startSlasher(t *testing.T, config *end2EndConfig) int {
	tmpPath := config.tmpPath
	binaryPath, found := bazel.FindBinary("slasher", "slasher")
	if !found {
		t.Log(binaryPath)
		t.Fatal("Slasher binary not found")
	}

	stdOutFile, err := deleteAndCreateFile(tmpPath, slasherLogFileName)
	if err != nil {
		t.Fatal(err)
	}

	args := []string{
		"--force-clear-db",
		"--span-map-cache",
		"--beacon-rpc-provider=4200",
		fmt.Sprintf("--datadir=%s/slasher", tmpPath),
		fmt.Sprintf("--log-file=%s", stdOutFile.Name()),
	}

	t.Logf("Starting slasher with flags: %s", strings.Join(args, " "))
	cmd := exec.Command(binaryPath, args...)
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start slasher client: %v", err)
	}

	if err = waitForTextInFile(stdOutFile, "Beacon node is fully synced, starting slashing detection"); err != nil {
		t.Fatalf("could not find starting logs for slasher, this means it had issues starting: %v", err)
	}

	return cmd.Process.Pid
}
