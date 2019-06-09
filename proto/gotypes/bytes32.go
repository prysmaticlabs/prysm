package gotypes

import "bytes"

type Bytes32 [32]byte

func NewBytes32(data []byte) *Bytes32 {
	b := &Bytes32{}
	_ = b.Unmarshal(data)
	return b
}

func (b Bytes32) Marshal() ([]byte, error) {
	buffer := make([]byte, 32)
	_, err := b.MarshalTo(buffer)
	return buffer, err
}

func (b Bytes32) MarshalTo(data []byte) (n int, err error) {
	copy(data, b[:])
	return b.Size(), nil
}

func (b Bytes32) Size() int {
	return 32
}

func (b *Bytes32) Unmarshal(data []byte) error {
	if data == nil || len(data) == 0 {
		*b = [32]byte{}
		return nil
	}
	copy(b[:], data)

	return nil
}

// TODO: test equality!
func (b Bytes32) Equal(data []byte) bool {
	return bytes.Equal(b[:], data)
}

func (this Bytes32) Compare(that Bytes96) int {
	return bytes.Compare(this[:], that[:])
}
