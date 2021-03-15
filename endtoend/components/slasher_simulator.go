package components

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/prysmaticlabs/prysm/endtoend/helpers"
	e2e "github.com/prysmaticlabs/prysm/endtoend/params"
)

func StartSlasherSimulator(t *testing.T) {
	binaryPath, found := bazel.FindBinary("cmd/beacon-chain", "beacon-chain")
	if !found {
		t.Log(binaryPath)
		t.Fatal("Beacon chain binary not found")
	}

	stdOutFile, err := helpers.DeleteAndCreateFile(e2e.TestParams.LogPath, e2e.SlasherSimulatorLogFileName)
	if err != nil {
		t.Fatal(err)
	}

	args := []string{
		"slasher-simulator",
		fmt.Sprintf("--datadir=%s/slasher-simulator-data/", e2e.TestParams.TestPath),
		fmt.Sprintf("--log-file=%s", stdOutFile.Name()),
		"--force-clear-db",
	}

	t.Logf("Starting slasher simulator with flags: %s", strings.Join(args, " "))
	cmd := exec.Command(binaryPath, args...)
	if err = cmd.Start(); err != nil {
		t.Fatalf("Failed to start slasher simulator client: %v", err)
	}

	if err = helpers.WaitForTextInFile(stdOutFile, "Producing blocks for slot 0"); err != nil {
		t.Fatalf("could not find starting logs for slasher simulator, this means it had issues starting: %v", err)
	}
}
