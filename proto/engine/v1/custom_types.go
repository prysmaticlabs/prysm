package enginev1

type BytesList [][]byte
type Bytes []byte
type Quantity uint64

func (b BytesList) MarshalJSON() ([]byte, error) {
	return nil, nil
}

func (b BytesList) UnmarshalJSON(enc []byte) error {
	return nil
}

func (b Bytes) MarshalJSON() ([]byte, error) {
	return nil, nil
}

func (b Bytes) UnmarshalJSON(enc []byte) error {
	return nil
}

func (q Quantity) MarshalJSON() ([]byte, error) {
	return nil, nil
}

func (q Quantity) UnmarshalJSON(enc []byte) error {
	return nil
}
