package decode

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshalUint8(t *testing.T) {
	var tests = []struct {
		in byte
		expected uint8
	}{
		{in: byte(8), expected: uint8(8)},
		{in: byte(0), expected: uint8(0)},
		{in: byte(255), expected: uint8(255)},
	}

	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert.Equal(t, tt.expected, UnmarshalUint8(tt.in))
		})
	}
}

func TestUnmarshalUint16(t *testing.T) {
	var tests = []struct {
		in []byte
		expected uint16
	}{
		{in: []byte{8, 0}, expected: uint16(8)},
		{in: []byte{0, 0}, expected: uint16(0)},
		{in: []byte{255, 255}, expected: uint16(65535)},
	}

	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			actual, err := UnmarshalUint16(tt.in)
			if err != nil {
				t.Error("UnmarshalUint16 returned error", err)
			}
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestUnmarshalUint32(t *testing.T) {
	var tests = []struct {
		in []byte
		expected uint32
	}{
		{in: []byte{8, 0, 0, 0}, expected: uint32(8)},
		{in: []byte{0, 0, 0, 0}, expected: uint32(0)},
		{in: []byte{255, 255, 255, 255}, expected: uint32(4294967295)},
	}

	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			actual, err := UnmarshalUint32(tt.in)
			if err != nil {
				t.Error("UnmarshalUint32 returned error", err)
			}
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestUnmarshalUint64(t *testing.T) {
	var tests = []struct {
		in []byte
		expected uint64
	}{
		{in: []byte{8, 0, 0, 0, 0, 0, 0, 0}, expected: uint64(8)},
		{in: []byte{0, 0, 0, 0, 0, 0, 0, 0}, expected: uint64(0)},
		{in: []byte{255, 255, 255, 255, 255, 255, 255, 255}, expected: uint64(18446744073709551615)},
	}

	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			actual, err := UnmarshalUint64(tt.in)
			if err != nil {
				t.Error("UnmarshalUint64 returned error", err)
			}
			assert.Equal(t, tt.expected, actual)
		})
	}
}
