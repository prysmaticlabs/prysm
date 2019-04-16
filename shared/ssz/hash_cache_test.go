package ssz

import (
	"bytes"
	"log"
	"reflect"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
)

type junkObject struct {
	D2Int64Slice [][]uint64
	Uint         uint64
	Int64Slice   []uint64
}

type tree struct {
	First  []*junkObject
	Second []*junkObject
}

func generateJunkObject(size uint64) []*junkObject {
	object := make([]*junkObject, size)
	for i := uint64(0); i < uint64(len(object)); i++ {
		d2Int64Slice := make([][]uint64, size)
		is := make([]uint64, size)
		uInt := uint64(time.Now().UnixNano())
		is[i] = i
		d2Int64Slice[i] = make([]uint64, size)
		for j := uint64(0); j < uint64(len(object)); j++ {
			d2Int64Slice[i][j] = i + j
		}
		object[i] = &junkObject{
			D2Int64Slice: d2Int64Slice,
			Uint:         uInt,
			Int64Slice:   is,
		}

	}
	return object
}

func TestObjCache_byHash(t *testing.T) {
	cache := newHashCache()
	byteSl := [][]byte{{0, 0}, {1, 1}}
	mr, err := merkleHash(byteSl)
	if err != nil {
		t.Fatal(err)
	}
	hs, err := hashedEncoding(reflect.ValueOf(byteSl))
	if err != nil {
		t.Fatal(err)
	}
	exists, _, err := cache.RootByEncodedHash(bytesutil.ToBytes32(hs))
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("Expected block info not to exist in empty cache")
	}
	if _, err := cache.MerkleHashCached(byteSl); err != nil {
		t.Fatal(err)
	}
	exists, fetchedInfo, err := cache.RootByEncodedHash(bytesutil.ToBytes32(hs))
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("Expected blockInfo to exist")
	}
	if !bytes.Equal(mr, fetchedInfo.MerkleRoot) {
		t.Errorf(
			"Expected fetched info number to be %v, got %v",
			mr,
			fetchedInfo.MerkleRoot,
		)
	}
	if fetchedInfo.Hash != bytesutil.ToBytes32(hs) {
		t.Errorf(
			"Expected fetched info hash to be %v, got %v",
			hs,
			fetchedInfo.Hash,
		)
	}
}

func TestMerkleHashWithCache(t *testing.T) {
	cache := newHashCache()
	for i := 0; i < 200; i++ {
		runMerkleHashTests(t, func(val [][]byte) ([]byte, error) {
			return merkleHash(val)
		})
	}
	for i := 0; i < 200; i++ {
		runMerkleHashTests(t, func(val [][]byte) ([]byte, error) {
			return cache.MerkleHashCached(val)
		})
	}
}

func BenchmarkHashWithoutCache(b *testing.B) {
	featureconfig.FeatureConfig().CacheTreeHash = false
	First := generateJunkObject(100)
	TreeHash(&tree{First: First, Second: First})
	for n := 0; n < b.N; n++ {
		TreeHash(&tree{First: First, Second: First})
	}
}

func BenchmarkHashWithCache(b *testing.B) {
	featureconfig.FeatureConfig().CacheTreeHash = true
	First := generateJunkObject(100)
	type tree struct {
		First  []*junkObject
		Second []*junkObject
	}
	TreeHash(&tree{First: First, Second: First})
	for n := 0; n < b.N; n++ {
		TreeHash(&tree{First: First, Second: First})
	}
}

func TestBlockCache_maxSize(t *testing.T) {
	maxCacheSize = 10000
	cache := newHashCache()
	for i := uint64(0); i < uint64(maxCacheSize+1025); i++ {

		if err := cache.AddRoot(bytesutil.ToBytes32(bytesutil.Bytes4(i)), []byte{1}); err != nil {
			t.Fatal(err)
		}
	}
	log.Printf(
		"hash cache key size is %d, itemcount is %d",
		maxCacheSize,
		cache.hashCache.ItemCount(),
	)
	time.Sleep(1 * time.Second)
	if int64(cache.hashCache.ItemCount()) > maxCacheSize {
		t.Errorf(
			"Expected hash cache key size to be %d, got %d",
			maxCacheSize,
			cache.hashCache.ItemCount(),
		)
	}
}
