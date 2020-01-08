package testing

import "github.com/prysmaticlabs/prysm/shared/slotutil"

var _ = slotutil.Ticker(&MockTicker{})
