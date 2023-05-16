package validator

import (
	"encoding/json"
)

// Epoch custom primitives.Epoch to be unmashalable
type Epoch uint64

func (e *Epoch) UnmarshalJSON(enc []byte) error {
	var val uint64
	err := json.Unmarshal(enc, &val)
	if err != nil {
		return err
	}
	*e = Epoch(val)
	return nil
}
