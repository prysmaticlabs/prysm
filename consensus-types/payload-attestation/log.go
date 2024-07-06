package payloadattestation

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

// logFields returns log fields for a ReadOnlyPayloadAtt instance.
func logFields(payload ReadOnlyPayloadAtt) logrus.Fields {
	return logrus.Fields{
		"slot":            payload.Slot(),
		"validatorIndex":  payload.ValidatorIndex(),
		"signature":       fmt.Sprintf("%#x", payload.Signature()),
		"beaconBlockRoot": fmt.Sprintf("%#x", payload.BeaconBlockRoot()),
		"payloadStatus":   payload.PayloadStatus(),
	}
}
