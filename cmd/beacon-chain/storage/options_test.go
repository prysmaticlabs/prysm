package storage

import (
	"flag"
	"testing"

	"github.com/prysmaticlabs/prysm/v4/cmd"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/urfave/cli/v2"
)

func TestBlobStoragePath_NoFlagSpecified(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String(cmd.DataDirFlag.Name, cmd.DataDirFlag.Value, cmd.DataDirFlag.Usage)
	cliCtx := cli.NewContext(&app, set, nil)
	storagePath := blobStoragePath(cliCtx)

	assert.Equal(t, cmd.DefaultDataDir()+"/blobs", storagePath)
}

func TestBlobStoragePath_FlagSpecified(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String(BlobStoragePath.Name, "/blah/blah", BlobStoragePath.Usage)
	cliCtx := cli.NewContext(&app, set, nil)
	storagePath := blobStoragePath(cliCtx)

	assert.Equal(t, "/blah/blah", storagePath)
}
