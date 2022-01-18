package kv

import (
	"context"
	"os"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/iface"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestStore_SaveGenesisData(t *testing.T) {
	ctx := context.Background()
	db := setupDB(t)

	gs, err := util.NewBeaconState()
	assert.NoError(t, err)

	assert.NoError(t, db.SaveGenesisData(ctx, gs))

	testGenesisDataSaved(t, db)
}

func testGenesisDataSaved(t *testing.T, db iface.Database) {
	ctx := context.Background()

	gb, err := db.GenesisBlock(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, gb)

	gbHTR, err := gb.Block().HashTreeRoot()
	assert.NoError(t, err)

	gss, err := db.StateSummary(ctx, gbHTR)
	assert.NoError(t, err)
	assert.NotNil(t, gss)

	head, err := db.HeadBlock(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, head)

	headHTR, err := head.Block().HashTreeRoot()
	assert.NoError(t, err)
	assert.Equal(t, gbHTR, headHTR, "head block does not match genesis block")
}

func TestLoadGenesisFromFile(t *testing.T) {
	fp := "testdata/mainnet.genesis.ssz"
	rfp, err := bazel.Runfile(fp)
	if err == nil {
		fp = rfp
	}

	r, err := os.Open(fp)
	assert.NoError(t, err)
	defer func() {
		assert.NoError(t, r.Close())
	}()

	db := setupDB(t)
	assert.NoError(t, db.LoadGenesis(context.Background(), r, false))
	testGenesisDataSaved(t, db)

	// Loading the same genesis again should not throw an error
	_, err = r.Seek(0, 0)
	assert.NoError(t, err)
	assert.NoError(t, db.LoadGenesis(context.Background(), r, false))
}

func TestLoadGenesisFromFile_mismatchedForkVersion(t *testing.T) {
	fp := "testdata/altona.genesis.ssz"
	rfp, err := bazel.Runfile(fp)
	if err == nil {
		fp = rfp
	}
	r, err := os.Open(fp)
	assert.NoError(t, err)
	defer func() {
		assert.NoError(t, r.Close())
	}()

	// Loading a genesis with the wrong fork version as beacon config should throw an error.
	db := setupDB(t)
	assert.ErrorContains(t, "does not match config genesis fork version", db.LoadGenesis(context.Background(), r, false))
}

func TestEnsureEmbeddedGenesis(t *testing.T) {
	// Embedded Genesis works with Mainnet config
	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig()
	cfg.ConfigName = params.ConfigNames[params.Mainnet]
	params.OverrideBeaconConfig(cfg)

	ctx := context.Background()
	db := setupDB(t)

	gb, err := db.GenesisBlock(ctx)
	assert.NoError(t, err)
	if gb != nil && !gb.IsNil() {
		t.Fatal("Genesis block exists already")
	}

	gs, err := db.GenesisState(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, gs, "an embedded genesis state does not exist")

	assert.NoError(t, db.EnsureEmbeddedGenesis(ctx))

	gb, err = db.GenesisBlock(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, gb)

	testGenesisDataSaved(t, db)
}
