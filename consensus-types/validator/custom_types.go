package validator

import (
	"encoding/json"

	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
)

// Epoch custom primitives.Epoch to be unmashalable
type Epoch primitives.Epoch

func (e *Epoch) UnmarshalJSON(enc []byte) error {
	var val uint64
	err := json.Unmarshal(enc, &val)
	if err != nil {
		return err
	}
	*e = Epoch(val)
	return nil
}
