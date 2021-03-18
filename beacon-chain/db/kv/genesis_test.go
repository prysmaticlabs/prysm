package kv

import (
	"context"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/iface"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
)

func TestStore_SaveGenesisData(t *testing.T) {
	ctx := context.Background()
	db := setupDB(t)

	gs, err := testutil.NewBeaconState()
	assert.NoError(t, err)

	assert.NoError(t, db.SaveGenesisData(ctx, gs))

	testGenesisDataSaved(t, db)
}

func testGenesisDataSaved(t *testing.T, db iface.Database) {
	ctx := context.Background()

	gb, err := db.GenesisBlock(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, gb)

	gbHTR, err := gb.Block.HashTreeRoot()
	assert.NoError(t, err)

	gss, err := db.StateSummary(ctx, gbHTR)
	assert.NoError(t, err)
	assert.NotNil(t, gss)

	head, err := db.HeadBlock(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, head)

	headHTR, err := head.Block.HashTreeRoot()
	assert.NoError(t, err)
	assert.Equal(t, gbHTR, headHTR, "head block does not match genesis block")

	jcp, err := db.JustifiedCheckpoint(ctx)
	assert.NoError(t, err)
	assert.Equal(t, bytesutil.ToBytes32(jcp.Root), gbHTR, "justified checkpoint not set to genesis block root")

	fcp, err := db.FinalizedCheckpoint(ctx)
	assert.NoError(t, err)
	assert.Equal(t, bytesutil.ToBytes32(fcp.Root), gbHTR, "finalized checkpoint not set to genesis block root")
}

func TestLoadGenesisFromFile(t *testing.T) {
	fp := "testdata/mainnet.genesis.ssz"
	rfp, err := bazel.Runfile(fp)
	if err == nil {
		fp = rfp
	}

	db := setupDB(t)
	assert.NoError(t, db.LoadGenesisFromFile(context.Background(), fp))
	testGenesisDataSaved(t, db)

	// Loading the same genesis again should not throw an error
	assert.NoError(t, db.LoadGenesisFromFile(context.Background(), fp))
}

func TestLoadGenesisFromFile_mismatchedForkVersion(t *testing.T) {
	fp := "testdata/altona.genesis.ssz"
	rfp, err := bazel.Runfile(fp)
	if err == nil {
		fp = rfp
	}

	// Loading a genesis with the wrong fork version as beacon config should throw an error.
	db := setupDB(t)
	assert.ErrorContains(t, "does not match config genesis fork version", db.LoadGenesisFromFile(context.Background(), fp))
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
	if gb != nil {
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
