package ssz_test

import (
	"reflect"
	"strconv"
	"testing"

	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/crypto/hash"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v3/encoding/ssz"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestUint64Root(t *testing.T) {
	uintVal := uint64(1234567890)
	expected := [32]byte{210, 2, 150, 73, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

	result := ssz.Uint64Root(uintVal)
	assert.Equal(t, expected, result)
}

func TestForkRoot(t *testing.T) {
	testFork := ethpb.Fork{
		PreviousVersion: []byte{123},
		CurrentVersion:  []byte{124},
		Epoch:           1234567890,
	}
	expected := [32]byte{19, 46, 77, 103, 92, 175, 247, 33, 100, 64, 17, 111, 199, 145, 69, 38, 217, 112, 6, 16, 149, 201, 225, 144, 192, 228, 197, 172, 157, 78, 114, 140}

	result, err := ssz.ForkRoot(&testFork)
	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestCheckPointRoot(t *testing.T) {
	testHasher := hash.CustomSHA256Hasher()
	testCheckpoint := ethpb.Checkpoint{
		Epoch: 1234567890,
		Root:  []byte{222},
	}
	expected := [32]byte{228, 65, 39, 109, 183, 249, 167, 232, 125, 239, 25, 155, 207, 4, 84, 174, 176, 229, 175, 224, 62, 33, 215, 254, 170, 220, 132, 65, 246, 128, 68, 194}

	result, err := ssz.CheckpointRoot(testHasher, &testCheckpoint)
	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestByteArrayRootWithLimit(t *testing.T) {
	testHistoricalRoots := [][]byte{{123}, {234}}
	expected := [32]byte{70, 204, 150, 196, 89, 138, 190, 205, 65, 207, 120, 166, 179, 247, 147, 20, 29, 133, 117, 116, 151, 234, 129, 32, 22, 15, 79, 178, 98, 73, 132, 152}

	result, err := ssz.ByteArrayRootWithLimit(testHistoricalRoots, 16777216)
	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestSlashingsRoot(t *testing.T) {
	testSlashingsRoot := []uint64{123, 234}
	expected := [32]byte{123, 0, 0, 0, 0, 0, 0, 0, 234, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

	result, err := ssz.SlashingsRoot(testSlashingsRoot)
	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestTransactionsRoot(t *testing.T) {
	tests := []struct {
		name    string
		txs     [][]byte
		want    [32]byte
		wantErr bool
	}{
		{
			name: "nil",
			txs:  nil,
			want: [32]byte{127, 254, 36, 30, 166, 1, 135, 253, 176, 24, 123, 250, 34, 222, 53, 209, 249, 190, 215, 171, 6, 29, 148, 1, 253, 71, 227, 74, 84, 251, 237, 225},
		},
		{
			name: "empty",
			txs:  [][]byte{},
			want: [32]byte{127, 254, 36, 30, 166, 1, 135, 253, 176, 24, 123, 250, 34, 222, 53, 209, 249, 190, 215, 171, 6, 29, 148, 1, 253, 71, 227, 74, 84, 251, 237, 225},
		},
		{
			name: "one tx",
			txs:  [][]byte{{1, 2, 3}},
			want: [32]byte{102, 209, 140, 87, 217, 28, 68, 12, 133, 42, 77, 136, 191, 18, 234, 105, 166, 228, 216, 235, 230, 95, 200, 73, 85, 33, 134, 254, 219, 97, 82, 209},
		},
		{
			name: "max txs",
			txs: func() [][]byte {
				var txs [][]byte
				for i := 0; i < fieldparams.MaxTxsPerPayloadLength; i++ {
					txs = append(txs, []byte{})
				}
				return txs
			}(),
			want: [32]byte{13, 66, 254, 206, 203, 58, 48, 133, 78, 218, 48, 231, 120, 90, 38, 72, 73, 137, 86, 9, 31, 213, 185, 101, 103, 144, 0, 236, 225, 57, 47, 244},
		},
		{
			name: "exceed max txs",
			txs: func() [][]byte {
				var txs [][]byte
				for i := 0; i < fieldparams.MaxTxsPerPayloadLength+1; i++ {
					txs = append(txs, []byte{})
				}
				return txs
			}(),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ssz.TransactionsRoot(tt.txs)
			if (err != nil) != tt.wantErr {
				t.Errorf("TransactionsRoot() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TransactionsRoot() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWithdrawalQueueRoot(t *testing.T) {
	const limit = 16
	tests := []struct {
		name    string
		ws      []*enginev1.Withdrawal
		want    [32]byte
		wantErr bool
	}{
		{
			name: "nil",
			ws:   nil,
			want: [32]byte{121, 41, 48, 187, 213, 186, 172, 67, 188, 199, 152, 238, 73, 170, 129, 133, 239, 118, 187, 59, 68, 186, 98, 185, 29, 134, 174, 86, 158, 75, 181, 53},
		},
		{
			name: "empty",
			ws:   []*enginev1.Withdrawal{},
			want: [32]byte{121, 41, 48, 187, 213, 186, 172, 67, 188, 199, 152, 238, 73, 170, 129, 133, 239, 118, 187, 59, 68, 186, 98, 185, 29, 134, 174, 86, 158, 75, 181, 53},
		},
		{
			name: "one",
			ws: []*enginev1.Withdrawal{{
				WithdrawalIndex:  123,
				ExecutionAddress: bytesutil.PadTo([]byte("address"), 20),
				Amount:           123,
			}},
			want: [32]byte{242, 16, 137, 49, 219, 35, 236, 39, 191, 96, 52, 104, 35, 98, 250, 177, 189, 65, 113, 185, 51, 107, 115, 26, 229, 168, 40, 189, 200, 152, 225, 24},
		},
		{
			name: "max withdrawals",
			ws: func() []*enginev1.Withdrawal {
				var ws []*enginev1.Withdrawal
				for i := 0; i < limit; i++ {
					ws = append(ws, &enginev1.Withdrawal{
						WithdrawalIndex:  uint64(i),
						ExecutionAddress: bytesutil.PadTo([]byte("address"+strconv.Itoa(i)), 20),
						Amount:           uint64(i),
					})
				}
				return ws
			}(),
			want: [32]byte{252, 43, 118, 69, 222, 73, 222, 196, 52, 234, 247, 49, 98, 54, 19, 146, 204, 246, 12, 139, 179, 167, 117, 216, 77, 159, 84, 154, 9, 103, 28, 226},
		},
		{
			name: "exceed max withdrawals",
			ws: func() []*enginev1.Withdrawal {
				var ws []*enginev1.Withdrawal
				for i := 0; i < limit+1; i++ {
					ws = append(ws, &enginev1.Withdrawal{
						WithdrawalIndex:  uint64(i),
						ExecutionAddress: bytesutil.PadTo([]byte("address"+strconv.Itoa(i)), 20),
						Amount:           uint64(i),
					})
				}
				return ws
			}(),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ssz.WithdrawalSliceRoot(hash.CustomSHA256Hasher(), tt.ws, limit)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPackByChunk_SingleList(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  [][32]byte
	}{
		{
			name:  "nil",
			input: nil,
			want:  [][32]byte{{}},
		},
		{
			name:  "empty",
			input: []byte{},
			want:  [][32]byte{{}},
		},
		{
			name:  "one",
			input: []byte{1},
			want:  [][32]byte{{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
		},
		{
			name:  "one, two",
			input: []byte{1, 2},
			want:  [][32]byte{{1, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ssz.PackByChunk([][]byte{tt.input})
			require.NoError(t, err)
			require.DeepSSZEqual(t, tt.want, got)
		})
	}
}
