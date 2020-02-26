package decode

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshalUint8(t *testing.T) {
	var tests = []struct {
		in       byte
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

func TestUnmarshalUint16_Successful(t *testing.T) {
	var tests = []struct {
		in       []byte
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

func TestUnmarshalUint16_Unsuccessful(t *testing.T) {
	var tests = []struct {
		in []byte
	}{
		{in: []byte{8, 0, 0, 0, 0, 0, 0, 0, 0}},
		{in: []byte{255}},
	}

	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			_, err := UnmarshalUint16(tt.in)
			assert.Error(t, err)
		})
	}
}

func TestUnmarshalUint32_Successful(t *testing.T) {
	var tests = []struct {
		in       []byte
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

func TestUnmarshalUint32_Unsuccessful(t *testing.T) {
	var tests = []struct {
		in []byte
	}{
		{in: []byte{8, 0, 0, 0, 0, 0, 0, 0, 0}},
		{in: []byte{255, 255, 255}},
	}

	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			_, err := UnmarshalUint32(tt.in)
			assert.Error(t, err)
		})
	}
}

func TestUnmarshalUint64_Successful(t *testing.T) {
	var tests = []struct {
		in       []byte
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

func TestUnmarshalUint64_Unsuccessful(t *testing.T) {
	var tests = []struct {
		in []byte
	}{
		{in: []byte{8, 0, 0, 0, 0, 0, 0, 0, 0}},
		{in: []byte{255, 255, 255, 255, 255, 255}},
	}

	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			_, err := UnmarshalUint64(tt.in)
			assert.Error(t, err)
		})
	}
}

func TestUnmarshalBoolean_Successful(t *testing.T) {
	var tests = []struct {
		in       byte
		expected bool
	}{
		{in: byte(1), expected: true},
		{in: byte(0), expected: false},
	}

	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			actual, err := UnmarshalBoolean(tt.in)
			if err != nil {
				t.Error(err)
			}
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestUnmarshalBoolean_Unsuccessful(t *testing.T) {
	var tests = []struct {
		in byte
	}{
		{in: byte(2)},
		{in: byte(255)},
	}

	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			_, err := UnmarshalBoolean(tt.in)
			assert.Error(t, err)
		})
	}
}
