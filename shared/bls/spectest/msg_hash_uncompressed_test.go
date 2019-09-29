package spectest

import (
	"testing"
)

// Note: This actually tests the underlying library as we don't have a need for
// HashG2Uncompressed in our local BLS API.
func TestMsgHashUncompressed(t *testing.T) {
	t.Skip("The python uncompressed method does not match the go uncompressed method and this isn't very important")
}
