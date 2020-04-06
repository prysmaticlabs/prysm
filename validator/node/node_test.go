package node

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	"gopkg.in/urfave/cli.v2"
)

// Test that the sharding node can build with default flag values.
func TestNode_Builds(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	tmpDir := testutil.TempDir()
	defer os.RemoveAll(tmpDir)
	set.String("datadir", filepath.Join(tmpDir, "datadir"), "the node data directory")
	set.String("keymanager", "interop", "key manager")
	set.String("keymanageropts", `{"keys":16,"offset":0}`, `key manager options`)
	set.String("verbosity", "debug", "log verbosity")
	context := cli.NewContext(&app, set, nil)

	_, err := NewValidatorClient(context)
	if err != nil {
		t.Fatalf("Failed to create ValidatorClient: %v", err)
	}
}
