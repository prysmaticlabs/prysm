package spectest

import (
	"testing"
)

func TestBlockHeaderMinimal(t *testing.T) {
	runBlockHeaderTest(t, "minimal")
}
