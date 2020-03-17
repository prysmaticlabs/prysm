package spectest

import (
	"testing"
)

func TestBlockHeaderMinimal(t *testing.T) {
	t.Skip("Skipping until last stage of 5119")
	runBlockHeaderTest(t, "minimal")
}
