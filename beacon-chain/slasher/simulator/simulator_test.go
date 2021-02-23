package simulator

import (
	"fmt"
	"testing"
)

func TestGenerateAttestations(t *testing.T) {
	simParams := &Parameters{
		NumValidators: 128,
	}
	atts := generateAttestationsForSlot(simParams, 0)

	for i, a := range atts {
		fmt.Println(i)
		fmt.Println(a)
	}
}
