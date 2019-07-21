package powchain

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
)

func TestHashKeyFn_OK(t *testing.T) {
	bInfo := &blockInfo{
		Hash: common.HexToHash("0x0123456"),
	}

	key, err := hashKeyFn(bInfo)
	if err != nil {
		t.Fatal(err)
	}
	if key != bInfo.Hash.Hex() {
		t.Errorf("Incorrect hash key: %s, expected %s", key, bInfo.Hash.Hex())
	}
}

func TestHashKeyFn_InvalidObj(t *testing.T) {
	_, err := hashKeyFn("bad")
	if err != ErrNotABlockInfo {
		t.Errorf("Expected error %v, got %v", ErrNotABlockInfo, err)
	}
}

func TestHeightKeyFn_OK(t *testing.T) {
	bInfo := &blockInfo{
		Number: big.NewInt(555),
	}

	key, err := heightKeyFn(bInfo)
	if err != nil {
		t.Fatal(err)
	}
	if key != bInfo.Number.String() {
		t.Errorf("Incorrect height key: %s, expected %s", key, bInfo.Number.String())
	}
}

func TestHeightKeyFn_InvalidObj(t *testing.T) {
	_, err := heightKeyFn("bad")
	if err != ErrNotABlockInfo {
		t.Errorf("Expected error %v, got %v", ErrNotABlockInfo, err)
	}
}

func TestBlockCache_byHash(t *testing.T) {
	cache := newBlockCache()

	header := &gethTypes.Header{
		ParentHash: common.HexToHash("0x12345"),
		Number:     big.NewInt(55),
	}

	exists, _, err := cache.BlockInfoByHash(header.Hash())
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("Expected block info not to exist in empty cache")
	}

	if err := cache.AddBlock(gethTypes.NewBlockWithHeader(header)); err != nil {
		t.Fatal(err)
	}

	exists, fetchedInfo, err := cache.BlockInfoByHash(header.Hash())
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("Expected blockInfo to exist")
	}
	if fetchedInfo.Number.Cmp(header.Number) != 0 {
		t.Errorf(
			"Expected fetched info number to be %v, got %v",
			header.Number,
			fetchedInfo.Number,
		)
	}
	if fetchedInfo.Hash != header.Hash() {
		t.Errorf(
			"Expected fetched info hash to be %v, got %v",
			header.Hash(),
			fetchedInfo.Hash,
		)
	}
}

func TestBlockCache_byHeight(t *testing.T) {
	cache := newBlockCache()

	header := &gethTypes.Header{
		ParentHash: common.HexToHash("0x12345"),
		Number:     big.NewInt(55),
	}

	exists, _, err := cache.BlockInfoByHeight(header.Number)
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("Expected block info not to exist in empty cache")
	}

	if err := cache.AddBlock(gethTypes.NewBlockWithHeader(header)); err != nil {
		t.Fatal(err)
	}

	exists, fetchedInfo, err := cache.BlockInfoByHeight(header.Number)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("Expected blockInfo to exist")
	}
	if fetchedInfo.Number.Cmp(header.Number) != 0 {
		t.Errorf(
			"Expected fetched info number to be %v, got %v",
			header.Number,
			fetchedInfo.Number,
		)
	}
	if fetchedInfo.Hash != header.Hash() {
		t.Errorf(
			"Expected fetched info hash to be %v, got %v",
			header.Hash(),
			fetchedInfo.Hash,
		)
	}
}

func TestBlockCache_maxSize(t *testing.T) {
	cache := newBlockCache()

	for i := int64(0); i < int64(maxCacheSize+10); i++ {
		header := &gethTypes.Header{
			Number: big.NewInt(i),
		}
		if err := cache.AddBlock(gethTypes.NewBlockWithHeader(header)); err != nil {
			t.Fatal(err)
		}
	}

	if len(cache.hashCache.ListKeys()) != maxCacheSize {
		t.Errorf(
			"Expected hash cache key size to be %d, got %d",
			maxCacheSize,
			len(cache.hashCache.ListKeys()),
		)
	}
	if len(cache.heightCache.ListKeys()) != maxCacheSize {
		t.Errorf(
			"Expected height cache key size to be %d, got %d",
			maxCacheSize,
			len(cache.heightCache.ListKeys()),
		)
	}
}
