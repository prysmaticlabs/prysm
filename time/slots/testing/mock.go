// Package testing includes useful mocks for slot tickers in unit tests.
package testing

import types "github.com/prysmaticlabs/eth2-types"

// MockTicker defines a useful struct for mocking the Ticker interface
// from the slotutil package.
type MockTicker struct {
	Channel chan types.Slot
}

// C --
func (m *MockTicker) C() <-chan types.Slot {
	return m.Channel
}

// Done --
func (_ *MockTicker) Done() {}
