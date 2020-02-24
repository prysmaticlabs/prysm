package encode

import (
	"strconv"
	"testing"

	"github.com/cockroachdb/cockroach/pkg/util/uint128"
	"github.com/stretchr/testify/assert"
)

func TestMarshalUint8(t *testing.T) {
	var tests = []struct {
		in uint8
		expected byte
	}{
		{in: uint8(8), expected: byte(8)},
		{in: uint8(0), expected: byte(0)},
		{in: uint8(255), expected: byte(255)},
	}

	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert.Equal(t, tt.expected, MarshalUint8(tt.in))
		})
	}
}

func TestMarshalUint16(t *testing.T) {
	var tests = []struct {
		in uint16
		expected []byte
	}{
		{in: uint16(8), expected: []byte{8, 0}},
		{in: uint16(0), expected: []byte{0, 0}},
		{in: uint16(65535), expected: []byte{255, 255}},
	}

	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert.Equal(t, tt.expected, MarshalUint16(tt.in))
		})
	}
}

func TestMarshalUint32(t *testing.T) {
	var tests = []struct {
		in uint32
		expected []byte
	}{
		{in: uint32(8), expected: []byte{8, 0, 0, 0}},
		{in: uint32(0), expected: []byte{0, 0, 0, 0}},
		{in: uint32(4294967295), expected: []byte{255, 255, 255, 255}},
	}

	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert.Equal(t, tt.expected, MarshalUint32(tt.in))
		})
	}
}

func TestMarshalUint64(t *testing.T) {
	var tests = []struct {
		in uint64
		expected []byte
	}{
		{in: uint64(8), expected: []byte{8, 0, 0, 0, 0, 0, 0, 0}},
		{in: uint64(0), expected: []byte{0, 0, 0, 0, 0, 0, 0, 0}},
		{in: uint64(18446744073709551615), expected: []byte{255, 255, 255, 255, 255, 255, 255, 255}},
	}

	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert.Equal(t, tt.expected, MarshalUint64(tt.in))
		})
	}
}

func TestMarshalUint128(t *testing.T) {
	var tests = []struct {
		in uint128.Uint128
		expected []byte
	}{
		{
			in: uint128.Uint128{8, 0},
			expected: []byte{8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			in: uint128.Uint128{0, 0},
			expected: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			in: uint128.Uint128{18446744073709551615, 18446744073709551615},
			expected: []byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
		},
	}

	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert.Equal(t, tt.expected, MarshalUint128(tt.in))
		})
	}
}


