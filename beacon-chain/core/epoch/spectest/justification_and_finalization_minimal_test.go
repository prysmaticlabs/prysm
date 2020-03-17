package spectest

import (
	"testing"
)

func TestJustificationAndFinalizationMinimal(t *testing.T) {
	t.Skip("Skipping until last stage of 5119")
	runJustificationAndFinalizationTests(t, "minimal")
}
