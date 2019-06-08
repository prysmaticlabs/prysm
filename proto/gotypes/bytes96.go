package gotypes

type Bytes96 [96]byte

func NewBytes96(data []byte) *Bytes96 {
	b := &Bytes96{}
	_ = b.Unmarshal(data)
	return b
}

func (b Bytes96) Marshal() ([]byte, error) {
	buffer := make([]byte, 96)
	_, err := b.MarshalTo(buffer)
	return buffer, err
}

func (b Bytes96) MarshalTo(data []byte) (n int, err error) {
	copy(data, b[:])
	return b.Size(), nil
}

func (b Bytes96) Size() int {
	return 96
}

func (b *Bytes96) Unmarshal(data []byte) error {
	if data == nil || len(data) == 0 {
		*b = [96]byte{}
		return nil
	}
	copy(b[:], data)

	return nil
}
