package forkchoice

import (
	"testing"
)

func TestMinimal_Deneb_Forkchoice(t *testing.T) {
	t.Skip("blocked by go-kzg-4844 minimal trusted setup")
	forkchoice.Run(t, "minimal", version.Deneb)
}
