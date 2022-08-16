package bytesutil_test

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestToBytes(t *testing.T) {
	tests := []struct {
		a uint64
		b []byte
	}{
		{0, []byte{0}},
		{1, []byte{1}},
		{2, []byte{2}},
		{253, []byte{253}},
		{254, []byte{254}},
		{255, []byte{255}},
		{0, []byte{0, 0}},
		{1, []byte{1, 0}},
		{255, []byte{255, 0}},
		{256, []byte{0, 1}},
		{65534, []byte{254, 255}},
		{65535, []byte{255, 255}},
		{0, []byte{0, 0, 0}},
		{255, []byte{255, 0, 0}},
		{256, []byte{0, 1, 0}},
		{65535, []byte{255, 255, 0}},
		{65536, []byte{0, 0, 1}},
		{16777215, []byte{255, 255, 255}},
		{0, []byte{0, 0, 0, 0}},
		{256, []byte{0, 1, 0, 0}},
		{65536, []byte{0, 0, 1, 0}},
		{16777216, []byte{0, 0, 0, 1}},
		{16777217, []byte{1, 0, 0, 1}},
		{4294967295, []byte{255, 255, 255, 255}},
		{0, []byte{0, 0, 0, 0, 0, 0, 0, 0}},
		{16777216, []byte{0, 0, 0, 1, 0, 0, 0, 0}},
		{4294967296, []byte{0, 0, 0, 0, 1, 0, 0, 0}},
		{4294967297, []byte{1, 0, 0, 0, 1, 0, 0, 0}},
		{9223372036854775806, []byte{254, 255, 255, 255, 255, 255, 255, 127}},
		{9223372036854775807, []byte{255, 255, 255, 255, 255, 255, 255, 127}},
	}
	for _, tt := range tests {
		b := bytesutil.ToBytes(tt.a, len(tt.b))
		assert.DeepEqual(t, tt.b, b)
	}
}

func TestBytes1(t *testing.T) {
	tests := []struct {
		a uint64
		b []byte
	}{
		{0, []byte{0}},
		{1, []byte{1}},
		{2, []byte{2}},
		{253, []byte{253}},
		{254, []byte{254}},
		{255, []byte{255}},
	}
	for _, tt := range tests {
		b := bytesutil.Bytes1(tt.a)
		assert.DeepEqual(t, tt.b, b)
	}
}

func TestBytes2(t *testing.T) {
	tests := []struct {
		a uint64
		b []byte
	}{
		{0, []byte{0, 0}},
		{1, []byte{1, 0}},
		{255, []byte{255, 0}},
		{256, []byte{0, 1}},
		{65534, []byte{254, 255}},
		{65535, []byte{255, 255}},
	}
	for _, tt := range tests {
		b := bytesutil.Bytes2(tt.a)
		assert.DeepEqual(t, tt.b, b)
	}
}

func TestBytes3(t *testing.T) {
	tests := []struct {
		a uint64
		b []byte
	}{
		{0, []byte{0, 0, 0}},
		{255, []byte{255, 0, 0}},
		{256, []byte{0, 1, 0}},
		{65535, []byte{255, 255, 0}},
		{65536, []byte{0, 0, 1}},
		{16777215, []byte{255, 255, 255}},
	}
	for _, tt := range tests {
		b := bytesutil.Bytes3(tt.a)
		assert.DeepEqual(t, tt.b, b)
	}
}

func TestBytes4(t *testing.T) {
	tests := []struct {
		a uint64
		b []byte
	}{
		{0, []byte{0, 0, 0, 0}},
		{256, []byte{0, 1, 0, 0}},
		{65536, []byte{0, 0, 1, 0}},
		{16777216, []byte{0, 0, 0, 1}},
		{16777217, []byte{1, 0, 0, 1}},
		{4294967295, []byte{255, 255, 255, 255}},
	}
	for _, tt := range tests {
		b := bytesutil.Bytes4(tt.a)
		assert.DeepEqual(t, tt.b, b)
	}
}

func TestBytes8(t *testing.T) {
	tests := []struct {
		a uint64
		b []byte
	}{
		{0, []byte{0, 0, 0, 0, 0, 0, 0, 0}},
		{16777216, []byte{0, 0, 0, 1, 0, 0, 0, 0}},
		{4294967296, []byte{0, 0, 0, 0, 1, 0, 0, 0}},
		{4294967297, []byte{1, 0, 0, 0, 1, 0, 0, 0}},
		{9223372036854775806, []byte{254, 255, 255, 255, 255, 255, 255, 127}},
		{9223372036854775807, []byte{255, 255, 255, 255, 255, 255, 255, 127}},
	}
	for _, tt := range tests {
		b := bytesutil.Bytes8(tt.a)
		assert.DeepEqual(t, tt.b, b)
	}
}

func TestFromBool(t *testing.T) {
	tests := []byte{
		0,
		1,
	}
	for _, tt := range tests {
		b := bytesutil.ToBool(tt)
		c := bytesutil.FromBool(b)
		assert.Equal(t, tt, c)
	}
}

func TestFromBytes2(t *testing.T) {
	tests := []uint64{
		0,
		1776,
		96726,
		(1 << 16) - 1,
	}
	for _, tt := range tests {
		b := bytesutil.ToBytes(tt, 2)
		c := bytesutil.FromBytes2(b)
		assert.Equal(t, uint16(tt), c)
	}
}

func TestFromBytes4(t *testing.T) {
	tests := []uint64{
		0,
		1776,
		96726,
		4290997,
		4294967295, // 2^32 - 1
		4294967200,
		3894948296,
	}
	for _, tt := range tests {
		b := bytesutil.ToBytes(tt, 4)
		c := bytesutil.FromBytes4(b)
		if c != tt {
			t.Errorf("Wanted %d but got %d", tt, c)
		}
		assert.Equal(t, tt, c)
	}
}

func TestFromBytes8(t *testing.T) {
	tests := []uint64{
		0,
		1776,
		96726,
		4290997,
		922376854775806,
		42893720984775807,
		18446744073709551615,
	}
	for _, tt := range tests {
		b := bytesutil.ToBytes(tt, 8)
		c := bytesutil.FromBytes8(b)
		assert.Equal(t, tt, c)
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		a []byte
		b []byte
	}{
		{[]byte{'A', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O'},
			[]byte{'A', 'C', 'D', 'E', 'F', 'G'}},
		{[]byte{'A', 'C', 'D', 'E', 'F'},
			[]byte{'A', 'C', 'D', 'E', 'F'}},
		{[]byte{}, []byte{}},
	}
	for _, tt := range tests {
		b := bytesutil.Trunc(tt.a)
		assert.DeepEqual(t, tt.b, b)
	}
}

func TestReverse(t *testing.T) {
	tests := []struct {
		input  [][32]byte
		output [][32]byte
	}{
		{[][32]byte{{'A'}, {'B'}, {'C'}, {'D'}, {'E'}, {'F'}, {'G'}, {'H'}},
			[][32]byte{{'H'}, {'G'}, {'F'}, {'E'}, {'D'}, {'C'}, {'B'}, {'A'}}},
		{[][32]byte{{1}, {2}, {3}, {4}},
			[][32]byte{{4}, {3}, {2}, {1}}},
		{[][32]byte{}, [][32]byte{}},
	}
	for _, tt := range tests {
		b := bytesutil.ReverseBytes32Slice(tt.input)
		assert.DeepEqual(t, tt.output, b)
	}
}

func TestSetBit(t *testing.T) {
	tests := []struct {
		a []byte
		b int
		c []byte
	}{
		{[]byte{0b00000000}, 1, []byte{0b00000010}},
		{[]byte{0b00000010}, 7, []byte{0b10000010}},
		{[]byte{0b10000010}, 9, []byte{0b10000010, 0b00000010}},
		{[]byte{0b10000010}, 27, []byte{0b10000010, 0b00000000, 0b00000000, 0b00001000}},
		{[]byte{0b10000010, 0b00000000}, 8, []byte{0b10000010, 0b00000001}},
		{[]byte{0b10000010, 0b00000000}, 31, []byte{0b10000010, 0b00000000, 0b00000000, 0b10000000}},
	}
	for _, tt := range tests {
		assert.DeepEqual(t, tt.c, bytesutil.SetBit(tt.a, tt.b))
	}
}

func TestClearBit(t *testing.T) {
	tests := []struct {
		a []byte
		b int
		c []byte
	}{
		{[]byte{0b00000000}, 1, []byte{0b00000000}},
		{[]byte{0b00000010}, 1, []byte{0b00000000}},
		{[]byte{0b10000010}, 1, []byte{0b10000000}},
		{[]byte{0b10000010}, 8, []byte{0b10000010}},
		{[]byte{0b10000010, 0b00001111}, 7, []byte{0b00000010, 0b00001111}},
		{[]byte{0b10000010, 0b00001111}, 10, []byte{0b10000010, 0b00001011}},
	}
	for _, tt := range tests {
		assert.DeepEqual(t, tt.c, bytesutil.ClearBit(tt.a, tt.b))
	}
}

func TestMakeEmptyBitfields(t *testing.T) {
	tests := []struct {
		a int
		b int
	}{
		{0, 1},
		{1, 1},
		{2, 1},
		{7, 1},
		{8, 2},
		{15, 2},
		{16, 3},
		{100, 13},
		{104, 14},
	}
	for _, tt := range tests {
		assert.DeepEqual(t, tt.b, len(bytesutil.MakeEmptyBitlists(tt.a)))
	}
}

func TestHighestBitIndex(t *testing.T) {
	tests := []struct {
		a     []byte
		b     int
		error bool
	}{
		{nil, 0, true},
		{[]byte{}, 0, true},
		{[]byte{0b00000001}, 1, false},
		{[]byte{0b10100101}, 8, false},
		{[]byte{0x00, 0x00}, 0, false},
		{[]byte{0xff, 0xa0}, 16, false},
		{[]byte{12, 34, 56, 78}, 31, false},
		{[]byte{255, 255, 255, 255}, 32, false},
	}
	for _, tt := range tests {
		i, err := bytesutil.HighestBitIndex(tt.a)
		if !tt.error {
			require.NoError(t, err)
			assert.DeepEqual(t, tt.b, i)
		} else {
			assert.ErrorContains(t, "input list can't be empty or nil", err)
		}
	}
}

func TestHighestBitIndexBelow(t *testing.T) {
	tests := []struct {
		a     []byte
		b     int
		c     int
		error bool
	}{
		{nil, 0, 0, true},
		{[]byte{}, 0, 0, true},
		{[]byte{0b00010001}, 0, 0, false},
		{[]byte{0b00010001}, 1, 1, false},
		{[]byte{0b00010001}, 2, 1, false},
		{[]byte{0b00010001}, 4, 1, false},
		{[]byte{0b00010001}, 5, 5, false},
		{[]byte{0b00010001}, 8, 5, false},
		{[]byte{0b00010001, 0b00000000}, 0, 0, false},
		{[]byte{0b00010001, 0b00000000}, 1, 1, false},
		{[]byte{0b00010001, 0b00000000}, 2, 1, false},
		{[]byte{0b00010001, 0b00000000}, 4, 1, false},
		{[]byte{0b00010001, 0b00000000}, 5, 5, false},
		{[]byte{0b00010001, 0b00000000}, 8, 5, false},
		{[]byte{0b00010001, 0b00000000}, 15, 5, false},
		{[]byte{0b00010001, 0b00000000}, 16, 5, false},
		{[]byte{0b00010001, 0b00100010}, 8, 5, false},
		{[]byte{0b00010001, 0b00100010}, 9, 5, false},
		{[]byte{0b00010001, 0b00100010}, 10, 10, false},
		{[]byte{0b00010001, 0b00100010}, 11, 10, false},
		{[]byte{0b00010001, 0b00100010}, 14, 14, false},
		{[]byte{0b00010001, 0b00100010}, 15, 14, false},
		{[]byte{0b00010001, 0b00100010}, 24, 14, false},
		{[]byte{0b00010001, 0b00100010, 0b10000000}, 23, 14, false},
		{[]byte{0b00010001, 0b00100010, 0b10000000}, 24, 24, false},
		{[]byte{0b00000000, 0b00000001, 0b00000011}, 17, 17, false},
		{[]byte{0b00000000, 0b00000001, 0b00000011}, 18, 18, false},
		{[]byte{12, 34, 56, 78}, 1000, 31, false},
		{[]byte{255, 255, 255, 255}, 1000, 32, false},
	}
	for _, tt := range tests {
		i, err := bytesutil.HighestBitIndexAt(tt.a, tt.b)
		if !tt.error {
			require.NoError(t, err)
			assert.DeepEqual(t, tt.c, i)
		} else {
			assert.ErrorContains(t, "input list can't be empty or nil", err)
		}
	}
}

func TestUint64ToBytes_RoundTrip(t *testing.T) {
	for i := uint64(0); i < 10000; i++ {
		b := bytesutil.Uint64ToBytesBigEndian(i)
		if got := bytesutil.BytesToUint64BigEndian(b); got != i {
			t.Error("Round trip did not match original value")
		}
	}
}

func TestIsHex(t *testing.T) {
	tests := []struct {
		a []byte
		b bool
	}{
		{nil, false},
		{[]byte(""), false},
		{[]byte("0x"), false},
		{[]byte("0x0"), true},
		{[]byte("foo"), false},
		{[]byte("1234567890abcDEF"), false},
		{[]byte("XYZ4567890abcDEF1234567890abcDEF1234567890abcDEF1234567890abcDEF"), false},
		{[]byte("0x1234567890abcDEF1234567890abcDEF1234567890abcDEF1234567890abcDEF"), true},
		{[]byte("1234567890abcDEF1234567890abcDEF1234567890abcDEF1234567890abcDEF"), false},
	}
	for _, tt := range tests {
		isHex := bytesutil.IsHex(tt.a)
		assert.Equal(t, tt.b, isHex)
	}
}

func TestSafeCopyRootAtIndex(t *testing.T) {
	tests := []struct {
		name    string
		input   [][]byte
		idx     uint64
		want    []byte
		wantErr bool
	}{
		{
			name:    "index out of range in non-empty slice",
			input:   [][]byte{{0x1}, {0x2}},
			idx:     2,
			wantErr: true,
		},
		{
			name:    "index out of range in empty slice",
			input:   [][]byte{},
			idx:     0,
			wantErr: true,
		},
		{
			name:  "nil input",
			input: nil,
			idx:   3,
			want:  nil,
		},
		{
			name:  "correct copy",
			input: [][]byte{{0x1}, {0x2}},
			idx:   1,
			want:  bytesutil.PadTo([]byte{0x2}, 32),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := bytesutil.SafeCopyRootAtIndex(tt.input, tt.idx)
			if (err != nil) != tt.wantErr {
				t.Errorf("SafeCopyRootAtIndex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SafeCopyRootAtIndex() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSafeCopy2dBytes(t *testing.T) {
	tests := []struct {
		name  string
		input [][]byte
	}{
		{
			name:  "nil input",
			input: nil,
		},
		{
			name:  "correct copy",
			input: [][]byte{{0x1}, {0x2}},
		},
		{
			name:  "empty",
			input: [][]byte{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := bytesutil.SafeCopy2dBytes(tt.input); !reflect.DeepEqual(got, tt.input) {
				t.Errorf("SafeCopy2dBytes() = %v, want %v", got, tt.input)
			}
		})
	}
}

func TestBytesInvalidInputs(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Test panicked: %v", r)
		}
	}()
	rawBytes := bytesutil.ToBytes(100, -10)
	assert.DeepEqual(t, []byte{}, rawBytes)

	_, err := bytesutil.HighestBitIndexAt([]byte{'A', 'B', 'C'}, -5)
	assert.ErrorContains(t, "index is negative", err)

	// There should be no panic
	_ = bytesutil.ClearBit([]byte{'C', 'D', 'E'}, -7)
	res := bytesutil.FromBytes4([]byte{})
	assert.Equal(t, res, uint64(0))
	newRes := bytesutil.FromBytes2([]byte{})
	assert.Equal(t, newRes, uint16(0))
	res = bytesutil.FromBytes8([]byte{})
	assert.Equal(t, res, uint64(0))

	intRes := bytesutil.ToLowInt64([]byte{})
	assert.Equal(t, intRes, int64(0))
}

func TestReverseByteOrder(t *testing.T) {
	input := []byte{0, 1, 2, 3, 4, 5}
	expectedResult := []byte{5, 4, 3, 2, 1, 0}
	output := bytesutil.ReverseByteOrder(input)

	// check that the input is not modified and the output is reversed
	assert.Equal(t, bytes.Equal(input, []byte{0, 1, 2, 3, 4, 5}), true)
	assert.Equal(t, bytes.Equal(expectedResult, output), true)
}

func TestSafeCopy2d32Bytes(t *testing.T) {
	input := make([][32]byte, 2)
	input[0] = bytesutil.ToBytes32([]byte{'a'})
	input[1] = bytesutil.ToBytes32([]byte{'b'})
	output := bytesutil.SafeCopy2d32Bytes(input)
	assert.Equal(t, false, &input == &output, "No copy was made")
	assert.DeepEqual(t, input, output)
}

func TestZeroRoot(t *testing.T) {
	input := make([]byte, fieldparams.RootLength)
	output := bytesutil.ZeroRoot(input)
	assert.Equal(t, true, output)
	copy(input[2:], "a")
	copy(input[3:], "b")
	output = bytesutil.ZeroRoot(input)
	assert.Equal(t, false, output)
}

func TestIsRoot(t *testing.T) {
	input := make([]byte, fieldparams.RootLength)
	output := bytesutil.IsRoot(input)
	assert.Equal(t, true, output)
}

func TestIsValidRoot(t *testing.T) {

	zeroRoot := make([]byte, fieldparams.RootLength)

	validRoot := make([]byte, fieldparams.RootLength)
	validRoot[0] = 'a'

	wrongLengthRoot := make([]byte, fieldparams.RootLength-4)
	wrongLengthRoot[0] = 'a'

	type args struct {
		root []byte
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Is ZeroRoot",
			args: args{
				root: zeroRoot,
			},
			want: false,
		},
		{
			name: "Is ValidRoot",
			args: args{
				root: validRoot,
			},
			want: true,
		},
		{
			name: "Is NonZeroRoot but not length 32",
			args: args{
				root: wrongLengthRoot,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bytesutil.IsValidRoot(tt.args.root)
			require.Equal(t, got, tt.want)
		})
	}
}

func TestUint32ToBytes4(t *testing.T) {
	tests := []struct {
		value uint32
		want  [4]byte
	}{
		{
			value: 0x01000000,
			want:  [4]byte{1, 0, 0, 0},
		},
		{
			value: 0x00000001,
			want:  [4]byte{0, 0, 0, 1},
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("0x%08x", tt.value), func(t *testing.T) {
			if got := bytesutil.Uint32ToBytes4(tt.value); !bytes.Equal(got[:], tt.want[:]) {
				t.Errorf("Uint32ToBytes4() = %v, want %v", got, tt.want)
			}
		})
	}
}
