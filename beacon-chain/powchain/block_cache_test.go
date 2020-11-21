package powchain

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestHashKeyFn_OK(t *testing.T) {
	hInfo := &headerInfo{
		Hash: common.HexToHash("0x0123456"),
	}

	key, err := hashKeyFn(hInfo)
	require.NoError(t, err)
	assert.Equal(t, hInfo.Hash.Hex(), key)
}

func TestHashKeyFn_InvalidObj(t *testing.T) {
	_, err := hashKeyFn("bad")
	assert.Equal(t, ErrNotAHeaderInfo, err)
}

func TestHeightKeyFn_OK(t *testing.T) {
	hInfo := &headerInfo{
		Number: big.NewInt(555),
	}

	key, err := heightKeyFn(hInfo)
	require.NoError(t, err)
	assert.Equal(t, hInfo.Number.String(), key)
}

func TestHeightKeyFn_InvalidObj(t *testing.T) {
	_, err := heightKeyFn("bad")
	assert.Equal(t, ErrNotAHeaderInfo, err)
}

func TestBlockCache_byHash(t *testing.T) {
	cache := newHeaderCache()

	header := &gethTypes.Header{
		ParentHash: common.HexToHash("0x12345"),
		Number:     big.NewInt(55),
	}

	exists, _, err := cache.HeaderInfoByHash(header.Hash())
	require.NoError(t, err)
	assert.Equal(t, false, exists, "Expected block info not to exist in empty cache")

	err = cache.AddHeader(header)
	require.NoError(t, err)

	exists, fetchedInfo, err := cache.HeaderInfoByHash(header.Hash())
	require.NoError(t, err)
	assert.Equal(t, true, exists, "Expected headerInfo to exist")
	assert.Equal(t, 0, fetchedInfo.Number.Cmp(header.Number), "Expected fetched info number to be equal")
	assert.Equal(t, header.Hash(), fetchedInfo.Hash, "Expected hash to be equal")

}

func TestBlockCache_byHeight(t *testing.T) {
	cache := newHeaderCache()

	header := &gethTypes.Header{
		ParentHash: common.HexToHash("0x12345"),
		Number:     big.NewInt(55),
	}

	exists, _, err := cache.HeaderInfoByHeight(header.Number)
	require.NoError(t, err)
	assert.Equal(t, false, exists, "Expected block info not to exist in empty cache")

	err = cache.AddHeader(header)
	require.NoError(t, err)

	exists, fetchedInfo, err := cache.HeaderInfoByHeight(header.Number)
	require.NoError(t, err)
	assert.Equal(t, true, exists, "Expected headerInfo to exist")

	assert.Equal(t, 0, fetchedInfo.Number.Cmp(header.Number), "Expected fetched info number to be equal")
	assert.Equal(t, header.Hash(), fetchedInfo.Hash, "Expected hash to be equal")

}

func TestBlockCache_maxSize(t *testing.T) {
	cache := newHeaderCache()

	for i := int64(0); i < int64(maxCacheSize+10); i++ {
		header := &gethTypes.Header{
			Number: big.NewInt(i),
		}
		err := cache.AddHeader(header)
		require.NoError(t, err)

	}

	assert.Equal(t, int(maxCacheSize), len(cache.hashCache.ListKeys()))
	assert.Equal(t, int(maxCacheSize), len(cache.heightCache.ListKeys()))
}

func Test_headerToHeaderInfo(t *testing.T) {
	type args struct {
		hdr *gethTypes.Header
	}
	tests := []struct {
		name    string
		args    args
		want    *headerInfo
		wantErr bool
	}{
		{
			name: "OK",
			args: args{hdr: &gethTypes.Header{
				Number: big.NewInt(500),
				Time:   2345,
			}},
			want: &headerInfo{
				Number: big.NewInt(500),
				Hash:   common.Hash{239, 10, 13, 71, 156, 192, 23, 93, 73, 154, 255, 209, 163, 204, 129, 12, 179, 183, 65, 70, 205, 200, 57, 12, 17, 211, 209, 4, 104, 133, 73, 86},
				Time:   2345,
			},
		},
		{
			name: "nil number",
			args: args{hdr: &gethTypes.Header{
				Time: 2345,
			}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := headerToHeaderInfo(tt.args.hdr)
			if (err != nil) != tt.wantErr {
				t.Errorf("headerToHeaderInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("headerToHeaderInfo() got = %v, want %v", got, tt.want)
			}
		})
	}
}
