package testing

import (
	"github.com/prysmaticlabs/prysm/time/slots"
)

var _ slots.Ticker = (*MockTicker)(nil)
