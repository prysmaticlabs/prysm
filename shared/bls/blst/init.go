package blst

import (
	"math"
	"runtime"

	blst "github.com/supranational/blst/bindings/go"
)

func init() {
	maxProcs := int(math.Ceil(float64(runtime.GOMAXPROCS(0)) * 0.75))
	if maxProcs <= 0 {
		maxProcs = 1
	}
	blst.SetMaxProcs(maxProcs)
}
