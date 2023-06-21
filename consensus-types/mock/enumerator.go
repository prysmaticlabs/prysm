package mock

// ZeroEnumerator always returns zero.
type ZeroEnumerator struct{}

// Inc --
func (c *ZeroEnumerator) Inc() uint64 {
	return 0
}
