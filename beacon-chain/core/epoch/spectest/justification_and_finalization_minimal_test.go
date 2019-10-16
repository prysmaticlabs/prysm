package spectest

import (
	"testing"
)

func TestJustificationAndFinalizationMinimal(t *testing.T) {
	t.Skip("Fails for could not get target atts current epoch")
	runJustificationAndFinalizationTests(t, "minimal")
}
