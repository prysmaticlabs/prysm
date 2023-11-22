package apimiddleware

import (
	"encoding/base64"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// EpochParticipation represents participation of validators in their duties.
type EpochParticipation []string

func (p *EpochParticipation) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}
	if len(b) < 2 {
		return errors.New("epoch participation length must be at least 2")
	}
	if b[0] != '"' || b[len(b)-1] != '"' {
		return errors.Errorf("provided epoch participation json string is malformed: %s", string(b))
	}

	// Remove leading and trailing quotation marks.
	jsonString := string(b)
	jsonString = strings.Trim(jsonString, "\"")
	decoded, err := base64.StdEncoding.DecodeString(jsonString)
	if err != nil {
		return errors.Wrapf(err, "could not decode epoch participation base64 value")
	}

	*p = make([]string, len(decoded))
	for i, participation := range decoded {
		(*p)[i] = strconv.FormatUint(uint64(participation), 10)
	}
	return nil
}
