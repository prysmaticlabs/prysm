package bitfield

import (
	"bytes"
	"testing"
)

func TestBitlist_Len(t *testing.T) {
	tests := []struct {
		bitlist Bitlist
		want    uint64
	}{
		{
			bitlist: Bitlist{},
			want:    0,
		},
		{
			bitlist: Bitlist{0x01}, // 0b00000001
			want:    0,
		},
		{
			bitlist: Bitlist{0x02}, // 0b00000010
			want:    1,
		},
		{
			bitlist: Bitlist{0x08}, // 0b00001000
			want:    3,
		},
		{
			bitlist: Bitlist{0x0E}, // 0b00001110
			want:    3,
		},
		{
			bitlist: Bitlist{0x0F}, // 0b00001111
			want:    3,
		},
		{
			bitlist: Bitlist{0x10}, // 0b00010000
			want:    4,
		},
		{
			bitlist: Bitlist{0x00, 0x01}, // 0b00000000, 0b00000001
			want:    8,
		},
		{
			bitlist: Bitlist{0x00, 0x02}, // 0b00000000, 0b00000010
			want:    9,
		},
		{
			bitlist: Bitlist{0x00, 0x02, 0x08}, // 0b00000000, 0b00000010, 0b00001000
			want:    19,
		},
	}

	for _, tt := range tests {
		if tt.bitlist.Len() != tt.want {
			t.Errorf("(%x).Len() = %d, wanted %d", tt.bitlist, tt.bitlist.Len(), tt.want)
		}
	}
}

func TestBitlist_BitAt(t *testing.T) {
	tests := []struct {
		bitlist Bitlist
		idx     uint64
		want    bool
	}{
		{
			bitlist: Bitlist{},
			idx:     0,
			want:    false,
		},
		{
			bitlist: Bitlist{0x01}, // 0b00000001
			idx:     55,            // Out of bounds
			want:    false,
		},
		{
			bitlist: Bitlist{0x01}, // 0b00000001
			idx:     0,             //          ^ (length bit)
			want:    false,
		},
		{
			bitlist: Bitlist{0x0E}, // 0b00001110
			idx:     0,             //          ^
			want:    false,
		},
		{
			bitlist: Bitlist{0x0E}, // 0b00001110
			idx:     1,             //         ^
			want:    true,
		},
		{
			bitlist: Bitlist{0x0E}, // 0b00001110
			idx:     3,             //       ^
			want:    false,
		},
		{
			bitlist: Bitlist{0x0E}, // 0b00001110
			idx:     4,             //       ^ (length bit)
			want:    false,
		},
		{
			bitlist: Bitlist{0xFF, 0x0F}, // 0b11111111, 0b00001111
			idx:     4,                   //      ^
			want:    true,
		},
		{
			bitlist: Bitlist{0xFF, 0x0F}, // 0b11111111, 0b00001111
			idx:     12,                  //                  ^
			want:    false,
		},
		{
			bitlist: Bitlist{0xFF, 0x0F}, // 0b11111111, 0b00001111
			idx:     11,                  //                   ^ (length bit)
			want:    false,
		},
		{
			bitlist: Bitlist{0x00, 0x0F}, // 0b00000000, 0b00001111
			idx:     10,                  //                    ^
			want:    true,
		},
	}

	for _, tt := range tests {
		if tt.bitlist.BitAt(tt.idx) != tt.want {
			t.Errorf(
				"(%x).BitAt(%d) = %t, wanted %t",
				tt.bitlist,
				tt.idx,
				tt.bitlist.BitAt(tt.idx),
				tt.want,
			)
		}
	}
}

func TestBitlist_SetBitAt(t *testing.T) {
	tests := []struct {
		bitlist Bitlist
		idx     uint64
		val     bool
		want    Bitlist
	}{
		{
			bitlist: Bitlist{},
			idx:     0,
			val:     true,
			want:    Bitlist{},
		},
		{
			bitlist: Bitlist{0x01}, // 0b00000001
			idx:     0,             //          ^
			val:     true,
			want:    Bitlist{0x01}, // 0b00000001
		},
		{
			bitlist: Bitlist{0x02}, // 0b00000010
			idx:     0,             //          ^
			val:     true,
			want:    Bitlist{0x03}, // 0b00000011
		},
		{
			bitlist: Bitlist{0x10}, // 0b00010000
			idx:     0,             //          ^
			val:     true,
			want:    Bitlist{0x11}, // 0b00010001
		},
		{
			bitlist: Bitlist{0x10}, // 0b00010000
			idx:     0,             //          ^
			val:     true,
			want:    Bitlist{0x11}, // 0b00010001
		},
		{
			bitlist: Bitlist{0x10}, // 0b00010000
			idx:     64,            // Out of bounds
			val:     true,
			want:    Bitlist{0x10}, // 0b00010001
		},
		{
			bitlist: Bitlist{0x1F}, // 0b00011111
			idx:     0,             //          ^
			val:     true,
			want:    Bitlist{0x1F}, // 0b00011111
		},
		{
			bitlist: Bitlist{0x1F}, // 0b00011111
			idx:     1,             //         ^
			val:     false,
			want:    Bitlist{0x1D}, // 0b00011101
		},
		{
			bitlist: Bitlist{0x1F}, // 0b00011111
			idx:     4,             //      ^ (length bit)
			val:     false,
			want:    Bitlist{0x1F}, // 0b00011111
		},
		{
			bitlist: Bitlist{0x1F}, // 0b00011111
			idx:     64,            // Out of bounds
			val:     false,
			want:    Bitlist{0x1F}, // 0b00011111
		},
		{
			bitlist: Bitlist{0x1F, 0x01}, // 0b00011111, 0b00000001
			idx:     0,                   //          ^
			val:     true,
			want:    Bitlist{0x1F, 0x01}, // 0b00011111, 0b00000001
		},
		{
			bitlist: Bitlist{0x1F, 0x01}, // 0b00011111, 0b00000001
			idx:     0,                   //          ^
			val:     false,
			want:    Bitlist{0x1E, 0x01}, // 0b00011110, 0b00000001
		},
		{
			bitlist: Bitlist{0x00, 0x10}, // 0b00000000, 0b00010000
			idx:     8,                   //                      ^
			val:     true,
			want:    Bitlist{0x00, 0x11}, // 0b00000000, 0b00010001
		},
		{
			bitlist: Bitlist{0x00, 0x11}, // 0b00000000, 0b00010001
			idx:     8,                   //                      ^
			val:     false,
			want:    Bitlist{0x00, 0x10}, // 0b00000000, 0b00010000
		},
	}

	for _, tt := range tests {
		original := make(Bitlist, len(tt.bitlist))
		copy(original, tt.bitlist)

		tt.bitlist.SetBitAt(tt.idx, tt.val)
		if !bytes.Equal(tt.bitlist, tt.want) {
			t.Errorf(
				"(%x).SetBitAt(%d, %t) = %x, wanted %x",
				original,
				tt.idx,
				tt.val,
				tt.bitlist,
				tt.want,
			)
		}
	}
}

func TestBitlist_Bytes(t *testing.T) {
	tests := []struct {
		bitlist Bitlist
		want    []byte
	}{
		{
			bitlist: Bitlist{0x01},
			want:    []byte{},
		},
		{
			bitlist: Bitlist{0x02},
			want:    []byte{},
		},
		{
			bitlist: Bitlist{0x03},
			want:    []byte{0x01},
		},
		{
			bitlist: Bitlist{0x12},
			want:    []byte{0x02},
		},
		{
			bitlist: Bitlist{0x02, 0x01},
			want:    []byte{0x02},
		},
		{
			bitlist: Bitlist{0x02, 0x02},
			want:    []byte{0x02},
		},
		{
			bitlist: Bitlist{0x02, 0x01},
			want:    []byte{0x02},
		},
		{
			bitlist: Bitlist{0x02, 0x03},
			want:    []byte{0x02, 0x01},
		},
		{
			bitlist: Bitlist{0x01, 0x00, 0x08},
			want:    []byte{0x01},
		}, {
			bitlist: Bitlist{0x00, 0x00, 0x02},
			want:    []byte{},
		},
	}

	for _, tt := range tests {
		got := tt.bitlist.Bytes()
		if !bytes.Equal(got, tt.want) {
			t.Errorf(
				"(%x).Bytes() = %x, wanted %x",
				tt.bitlist,
				got,
				tt.want,
			)
		}
	}
}
