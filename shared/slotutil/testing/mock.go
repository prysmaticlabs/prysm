// Package testing includes useful mocks for slot tickers in unit tests.
package testing

// MockTicker defines a useful struct for mocking the Ticker interface
// from the slotutil package.
type MockTicker struct {
	Channel chan uint64
}

// C --
func (m *MockTicker) C() <-chan uint64 {
	return m.Channel
}

// Done --
func (m *MockTicker) Done() {}
