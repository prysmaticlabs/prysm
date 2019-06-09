package gotypes

import "bytes"

type Bytes4 [4]byte

func NewBytes4(data []byte) *Bytes32 {
	b := &Bytes32{}
	_ = b.Unmarshal(data)
	return b
}

func (b Bytes4) Marshal() ([]byte, error) {
	buffer := make([]byte, 4)
	_, err := b.MarshalTo(buffer)
	return buffer, err
}

func (b Bytes4) MarshalTo(data []byte) (n int, err error) {
	copy(data, b[:])
	return b.Size(), nil
}

func (b Bytes4) Size() int {
	return 4
}

func (b *Bytes4) Unmarshal(data []byte) error {
	if data == nil || len(data) == 0 {
		*b = [4]byte{}
		return nil
	}
	copy(b[:], data)

	return nil
}

// TODO: test equality!
func (b Bytes4) Equal(data []byte) bool {
	return bytes.Equal(b[:], data)
}

func (this Bytes4) Compare(that Bytes96) int {
	return bytes.Compare(this[:], that[:])
}
