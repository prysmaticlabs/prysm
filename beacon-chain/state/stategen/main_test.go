package stategen

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state/state-native/types"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/mock"
)

func TestMain(m *testing.M) {
	types.Enumerator = &mock.ZeroEnumerator{}
	m.Run()
}
