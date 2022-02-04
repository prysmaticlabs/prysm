package enginev1

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
)

type BytesHash []byte
type Quantity uint64

func (b BytesHash) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%#x", b)), nil
}

func (b BytesHash) UnmarshalJSON(enc []byte) error {
	decoded, err := hex.DecodeString(strings.TrimPrefix(string(enc), "0x"))
	if err != nil {
		return err
	}
	b = decoded
	return nil
}

func (q Quantity) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%d", q)), nil
}

func (q Quantity) UnmarshalJSON(enc []byte) error {
	decoded, err := hex.DecodeString(strings.TrimPrefix(string(enc), "0x"))
	if err != nil {
		return err
	}
	q = Quantity(binary.LittleEndian.Uint64(decoded))
	return nil
}
