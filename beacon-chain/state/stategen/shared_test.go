package stategen

import (
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/db/iface"
)

type testSaverOpt func(h *hotStateSaver)

func withFinalizedCheckpointer(fc *mockFinalizedCheckpointer) testSaverOpt {
	return func(h *hotStateSaver) {
		h.fc = fc
	}
}

func newTestSaver(db iface.HeadAccessDatabase, opts ...testSaverOpt) *hotStateSaver {
	h := &hotStateSaver{
		snapshotInterval: DefaultSnapshotInterval,
		db:               db,
		fc:               &mockFinalizedCheckpointer{},
		cs:               &mockCurrentSlotter{},
	}
	for _, o := range opts {
		o(h)
	}

	return h
}
