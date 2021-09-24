package kv

import (
	"context"
	"crypto/rand"
	"testing"

	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	bolt "go.etcd.io/bbolt"
)

func Test_deleteValueForIndices(t *testing.T) {
	db := setupDB(t)
	blocks := make([]byte, 128)
	_, err := rand.Read(blocks)
	require.NoError(t, err)
	tests := []struct {
		name          string
		inputIndices  map[string][]byte
		root          []byte
		outputIndices map[string][]byte
		wantedErr     string
	}{
		{
			name:          "empty input, no root",
			inputIndices:  map[string][]byte{},
			root:          []byte{},
			outputIndices: map[string][]byte{},
		},
		{
			name:          "empty input, root does not exist",
			inputIndices:  map[string][]byte{},
			root:          bytesutil.PadTo([]byte("not found"), 32),
			outputIndices: map[string][]byte{},
		},
		{
			name: "non empty input, root does not exist",
			inputIndices: map[string][]byte{
				"blocks": bytesutil.PadTo([]byte{0xde, 0xad, 0xbe, 0xef}, 64),
			},
			root: bytesutil.PadTo([]byte("not found"), 32),
			outputIndices: map[string][]byte{
				"blocks": bytesutil.PadTo([]byte{0xde, 0xad, 0xbe, 0xef}, 64),
			},
		},
		{
			name: "removes value for a single bucket",
			inputIndices: map[string][]byte{
				"blocks": {0xde, 0xad, 0xbe, 0xef},
			},
			root: []byte{0xde},
			outputIndices: map[string][]byte{
				"blocks": {0xad, 0xbe, 0xef},
			},
		},
		{
			name: "removes multi-byte value for a single bucket (non-aligned)",
			inputIndices: map[string][]byte{
				"blocks": {0xde, 0xad, 0xbe, 0xef},
			},
			root: []byte{0xad, 0xbe},
			outputIndices: map[string][]byte{
				"blocks": {0xde, 0xad, 0xbe, 0xef},
			},
		},
		{
			name: "removes multi-byte value for a single bucket (non-aligned)",
			inputIndices: map[string][]byte{
				"blocks": {0xde, 0xad, 0xbe, 0xef, 0xff, 0x01},
			},
			root: []byte{0xbe, 0xef},
			outputIndices: map[string][]byte{
				"blocks": {0xde, 0xad, 0xff, 0x01},
			},
		},
		{
			name: "removes value from multiple buckets",
			inputIndices: map[string][]byte{
				"blocks":      {0xff, 0x32, 0x45, 0x25, 0xde, 0xad, 0xbe, 0xef, 0x24},
				"state":       {0x01, 0x02, 0x03, 0x04},
				"check-point": {0xde, 0xad, 0xbe, 0xef},
				"powchain":    {0xba, 0xad, 0xb0, 0x00, 0xde, 0xad, 0xbe, 0xef, 0xff},
			},
			root: []byte{0xde, 0xad, 0xbe, 0xef},
			outputIndices: map[string][]byte{
				"blocks":      {0xff, 0x32, 0x45, 0x25, 0x24},
				"state":       {0x01, 0x02, 0x03, 0x04},
				"check-point": nil,
				"powchain":    {0xba, 0xad, 0xb0, 0x00, 0xff},
			},
		},
		{
			name: "root as subsequence of two values (preserve)",
			inputIndices: map[string][]byte{
				"blocks": blocks,
			},
			outputIndices: map[string][]byte{
				"blocks": blocks,
			},
			root: blocks[48:80],
		},
		{
			name: "root as subsequence of two values (remove)",
			inputIndices: map[string][]byte{
				"blocks": blocks,
			},
			outputIndices: map[string][]byte{
				"blocks": append(blocks[0:64], blocks[96:]...),
			},
			root: blocks[64:96],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.db.Update(func(tx *bolt.Tx) error {
				for k, idx := range tt.inputIndices {
					bkt := tx.Bucket([]byte(k))
					require.NoError(t, bkt.Put(idx, tt.inputIndices[k]))
				}
				err := deleteValueForIndices(context.Background(), tt.inputIndices, tt.root, tx)
				if tt.wantedErr != "" {
					assert.ErrorContains(t, tt.wantedErr, err)
					return nil
				}
				assert.NoError(t, err)
				// Check updated indices.
				for k, idx := range tt.inputIndices {
					bkt := tx.Bucket([]byte(k))
					valuesAtIndex := bkt.Get(idx)
					assert.DeepEqual(t, tt.outputIndices[k], valuesAtIndex)
				}
				return nil
			})
			require.NoError(t, err)
		})
	}
}
