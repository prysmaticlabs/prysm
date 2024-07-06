package payloadattestation

import "github.com/pkg/errors"

var (
	errNilPayloadMessage       = errors.New("received nil payload message")
	errNilPayloadData          = errors.New("received nil payload data")
	errMissingPayloadSignature = errors.New("received nil payload signature")
	ErrMismatchCurrentSlot     = errors.New("does not match current slot")
	ErrUnknownPayloadStatus    = errors.New("unknown payload status")
)
