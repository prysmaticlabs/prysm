package bytesutil_test

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
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
		if !bytes.Equal(b, tt.b) {
			t.Errorf("Bytes1(%d) = %v, want = %d", tt.a, b, tt.b)
		}
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
		if !bytes.Equal(b, tt.b) {
			t.Errorf("Bytes1(%d) = %v, want = %d", tt.a, b, tt.b)
		}
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
		if !bytes.Equal(b, tt.b) {
			t.Errorf("Bytes2(%d) = %v, want = %d", tt.a, b, tt.b)
		}
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
		if !bytes.Equal(b, tt.b) {
			t.Errorf("Bytes3(%d) = %v, want = %d", tt.a, b, tt.b)
		}
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
		if !bytes.Equal(b, tt.b) {
			t.Errorf("Bytes4(%d) = %v, want = %d", tt.a, b, tt.b)
		}
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
		if !bytes.Equal(b, tt.b) {
			t.Errorf("Bytes8(%d) = %v, want = %d", tt.a, b, tt.b)
		}
	}
}

func TestFromBytes4(t *testing.T) {
	tests := []uint64{
		0,
		1776,
		96726,
		4290997,
		4294967295, //2^32 - 1
		4294967200,
		3894948296,
	}
	for _, tt := range tests {
		b := bytesutil.ToBytes(tt, 4)
		c := bytesutil.FromBytes4(b)
		if c != tt {
			t.Errorf("Wanted %d but got %d", tt, c)
		}
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
		if c != tt {
			t.Errorf("Wanted %d but got %d", tt, c)
		}
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
		if !bytes.Equal(b, tt.b) {
			t.Errorf("Trunc(%d) = %v, want = %d", tt.a, b, tt.b)
		}
	}
}

func TestReverse(t *testing.T) {
	tests := []struct {
		input  [][32]byte
		output [][32]byte
	}{
		{[][32]byte{[32]byte{'A'}, [32]byte{'B'}, [32]byte{'C'}, [32]byte{'D'}, [32]byte{'E'}, [32]byte{'F'}, [32]byte{'G'}, [32]byte{'H'}},
			[][32]byte{[32]byte{'H'}, [32]byte{'G'}, [32]byte{'F'}, [32]byte{'E'}, [32]byte{'D'}, [32]byte{'C'}, [32]byte{'B'}, [32]byte{'A'}}},
		{[][32]byte{[32]byte{1}, [32]byte{2}, [32]byte{3}, [32]byte{4}},
			[][32]byte{[32]byte{4}, [32]byte{3}, [32]byte{2}, [32]byte{1}}},
		{[][32]byte{}, [][32]byte{}},
	}
	for _, tt := range tests {
		b := bytesutil.ReverseBytes32Slice(tt.input)
		if !reflect.DeepEqual(b, tt.output) {
			t.Errorf("Reverse(%d) = %v, want = %d", tt.input, b, tt.output)
		}
	}
}
