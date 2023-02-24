package testing

import (
	"github.com/prysmaticlabs/prysm/v3/time/slots"
)

var _ slots.Ticker = (*MockTicker)(nil)
