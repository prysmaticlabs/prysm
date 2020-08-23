package powchain

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestHashKeyFn_OK(t *testing.T) {
	bInfo := &blockInfo{
		Hash: common.HexToHash("0x0123456"),
	}

	key, err := hashKeyFn(bInfo)
	require.NoError(t, err)
	assert.Equal(t, bInfo.Hash.Hex(), key)
}

func TestHashKeyFn_InvalidObj(t *testing.T) {
	_, err := hashKeyFn("bad")
	assert.Equal(t, ErrNotABlockInfo, err)
}

func TestHeightKeyFn_OK(t *testing.T) {
	bInfo := &blockInfo{
		Number: big.NewInt(555),
	}

	key, err := heightKeyFn(bInfo)
	require.NoError(t, err)
	assert.Equal(t, bInfo.Number.String(), key)
}

func TestHeightKeyFn_InvalidObj(t *testing.T) {
	_, err := heightKeyFn("bad")
	assert.Equal(t, ErrNotABlockInfo, err)
}

func TestBlockCache_byHash(t *testing.T) {
	cache := newBlockCache()

	header := &gethTypes.Header{
		ParentHash: common.HexToHash("0x12345"),
		Number:     big.NewInt(55),
	}

	exists, _, err := cache.BlockInfoByHash(header.Hash())
	require.NoError(t, err)
	assert.Equal(t, false, exists, "Expected block info not to exist in empty cache")

	err = cache.AddBlock(gethTypes.NewBlockWithHeader(header))
	require.NoError(t, err)

	exists, fetchedInfo, err := cache.BlockInfoByHash(header.Hash())
	require.NoError(t, err)
	assert.Equal(t, true, exists, "Expected blockInfo to exist")
	assert.Equal(t, 0, fetchedInfo.Number.Cmp(header.Number), "Expected fetched info number to be equal")
	assert.Equal(t, header.Hash(), fetchedInfo.Hash, "Expected hash to be equal")

}

func TestBlockCache_byHeight(t *testing.T) {
	cache := newBlockCache()

	header := &gethTypes.Header{
		ParentHash: common.HexToHash("0x12345"),
		Number:     big.NewInt(55),
	}

	exists, _, err := cache.BlockInfoByHeight(header.Number)
	require.NoError(t, err)
	assert.Equal(t, false, exists, "Expected block info not to exist in empty cache")

	err = cache.AddBlock(gethTypes.NewBlockWithHeader(header))
	require.NoError(t, err)

	exists, fetchedInfo, err := cache.BlockInfoByHeight(header.Number)
	require.NoError(t, err)
	assert.Equal(t, true, exists, "Expected blockInfo to exist")

	assert.Equal(t, 0, fetchedInfo.Number.Cmp(header.Number), "Expected fetched info number to be equal")
	assert.Equal(t, header.Hash(), fetchedInfo.Hash, "Expected hash to be equal")

}

func TestBlockCache_maxSize(t *testing.T) {
	cache := newBlockCache()

	for i := int64(0); i < int64(maxCacheSize+10); i++ {
		header := &gethTypes.Header{
			Number: big.NewInt(i),
		}
		err := cache.AddBlock(gethTypes.NewBlockWithHeader(header))
		require.NoError(t, err)

	}

	assert.Equal(t, int(maxCacheSize), len(cache.hashCache.ListKeys()))
	assert.Equal(t, int(maxCacheSize), len(cache.heightCache.ListKeys()))
}
