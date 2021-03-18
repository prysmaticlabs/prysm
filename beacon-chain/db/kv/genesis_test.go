package kv

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
)

func TestStore_SaveGenesisData(t *testing.T) {
	ctx := context.Background()
	db := setupDB(t)

	gs, err := testutil.NewBeaconState()
	assert.NoError(t, err)

	assert.NoError(t, db.SaveGenesisData(ctx, gs))

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
