// This file exists just to give me some bootstraping input to run through github.com/aloder/tojen
// to speed up the tedious process of writing jen code
package testdata


import (
	"github.com/prysmaticlabs/prysm/sszgen/types"
)


var testing types.ValRep = &types.ValueContainer{
	Name: "testing",
	Package: "github.com/prysmaticlabs/derp",
	Contents: map[string]types.ValRep{
		"OverlayUint": &types.ValuePointer{
			Referent: &types.ValueOverlay{
				Name: "FakeContainer",
				Package: "github.com/prysmaticlabs/derp/derp",
				Underlying: &types.ValueUint{
					Name: "uint8",
					Size: 8,
				},
			},
		},
	},
}
