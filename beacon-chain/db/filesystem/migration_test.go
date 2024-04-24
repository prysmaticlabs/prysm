package filesystem

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
	"github.com/spf13/afero"
)

func testSetupPaths(t *testing.T, fs afero.Fs, paths []migrateBeforeAfter) {
	for _, ba := range paths {
		slot, err := slots.EpochStart(ba.epoch)
		require.NoError(t, err)
		slot += ba.slotOffset
		_, sc := util.GenerateTestDenebBlockWithSidecar(t, [32]byte{}, slot, 1)
		scb, err := sc[0].MarshalSSZ()
		require.NoError(t, err)
		p := ba.before
		dir := filepath.Dir(p)
		require.NoError(t, fs.MkdirAll(dir, directoryPermissions))
		require.NoError(t, afero.WriteFile(fs, p, scb, 0666))
		_, err = fs.Stat(ba.before)
		require.NoError(t, err)
	}
}

func testAssertNewPaths(t *testing.T, fs afero.Fs, bs *BlobStorage, paths []migrateBeforeAfter) {
	for _, ba := range paths {
		if ba.before != ba.after {
			_, err := fs.Stat(ba.before)
			require.ErrorIs(t, err, os.ErrNotExist)
			dir := filepath.Dir(ba.before)
			_, err = listDir(fs, dir)
			require.ErrorIs(t, err, os.ErrNotExist)
		}
		_, err := fs.Stat(ba.after)
		require.NoError(t, err)
		root, err := stringToRoot(ba.root)
		require.NoError(t, err)
		namer, err := bs.layout.ident(root, ba.index)
		require.NoError(t, err)
		path := bs.layout.sszPath(namer)
		require.Equal(t, ba.after, path)
	}
}

type migrateBeforeAfter struct {
	before     string
	after      string
	epoch      primitives.Epoch
	slotOffset primitives.Slot
	index      uint64
	root       string
}

func TestPeriodicEpochMigrator(t *testing.T) {
	cases := []struct {
		name string
		plan []migrateBeforeAfter
		err  error
	}{
		{
			name: "happy path",
			plan: []migrateBeforeAfter{
				{
					before:     "0x0125e54c64c925018c9296965a5b622d9f5ab626c10917860dcfb6aa09a0a00b/0.ssz",
					epoch:      1234,
					slotOffset: 0,
					root:       "0x0125e54c64c925018c9296965a5b622d9f5ab626c10917860dcfb6aa09a0a00b",
					index:      0,
					after:      periodicEpochBaseDir + "/0/1234/0x0125e54c64c925018c9296965a5b622d9f5ab626c10917860dcfb6aa09a0a00b/0.ssz",
				},
				{
					before:     "0x0127dba6fd30fdbb47e73e861d5c6e602b38ac3ddc945bb6a2fc4e10761e9a86/0.ssz",
					root:       "0x0127dba6fd30fdbb47e73e861d5c6e602b38ac3ddc945bb6a2fc4e10761e9a86",
					index:      0,
					epoch:      5330,
					slotOffset: 0,
					after:      periodicEpochBaseDir + "/1/5330/0x0127dba6fd30fdbb47e73e861d5c6e602b38ac3ddc945bb6a2fc4e10761e9a86/0.ssz",
				},
				{
					before:     "0x0127dba6fd30fdbb47e73e861d5c6e602b38ac3ddc945bb6a2fc4e10761e9a86/1.ssz",
					root:       "0x0127dba6fd30fdbb47e73e861d5c6e602b38ac3ddc945bb6a2fc4e10761e9a86",
					index:      1,
					epoch:      5330,
					slotOffset: 31,
					after:      periodicEpochBaseDir + "/1/5330/0x0127dba6fd30fdbb47e73e861d5c6e602b38ac3ddc945bb6a2fc4e10761e9a86/1.ssz",
				},
				{
					before:     "0x0232521756a0b965eab2c2245d7ad85feaeaf5f427cd14d1a7531f9d555b415c/0.ssz",
					root:       "0x0232521756a0b965eab2c2245d7ad85feaeaf5f427cd14d1a7531f9d555b415c",
					index:      0,
					epoch:      16777216,
					slotOffset: 16,
					after:      periodicEpochBaseDir + "/4096/16777216/0x0232521756a0b965eab2c2245d7ad85feaeaf5f427cd14d1a7531f9d555b415c/0.ssz",
				},
			},
		},
		{
			name: "mix old and new",
			plan: []migrateBeforeAfter{
				{
					before:     "0x0125e54c64c925018c9296965a5b622d9f5ab626c10917860dcfb6aa09a0a00b/0.ssz",
					root:       "0x0125e54c64c925018c9296965a5b622d9f5ab626c10917860dcfb6aa09a0a00b",
					index:      0,
					epoch:      1234,
					slotOffset: 0,
					after:      periodicEpochBaseDir + "/0/1234/0x0125e54c64c925018c9296965a5b622d9f5ab626c10917860dcfb6aa09a0a00b/0.ssz",
				},
				{
					before:     "0x0127dba6fd30fdbb47e73e861d5c6e602b38ac3ddc945bb6a2fc4e10761e9a86/0.ssz",
					root:       "0x0127dba6fd30fdbb47e73e861d5c6e602b38ac3ddc945bb6a2fc4e10761e9a86",
					index:      0,
					epoch:      5330,
					slotOffset: 0,
					after:      periodicEpochBaseDir + "/1/5330/0x0127dba6fd30fdbb47e73e861d5c6e602b38ac3ddc945bb6a2fc4e10761e9a86/0.ssz",
				},
				{
					before:     "0x0127dba6fd30fdbb47e73e861d5c6e602b38ac3ddc945bb6a2fc4e10761e9a86/1.ssz",
					root:       "0x0127dba6fd30fdbb47e73e861d5c6e602b38ac3ddc945bb6a2fc4e10761e9a86",
					index:      1,
					epoch:      5330,
					slotOffset: 31,
					after:      periodicEpochBaseDir + "/1/5330/0x0127dba6fd30fdbb47e73e861d5c6e602b38ac3ddc945bb6a2fc4e10761e9a86/1.ssz",
				},
				{
					before:     "0x0232521756a0b965eab2c2245d7ad85feaeaf5f427cd14d1a7531f9d555b415c/0.ssz",
					root:       "0x0232521756a0b965eab2c2245d7ad85feaeaf5f427cd14d1a7531f9d555b415c",
					index:      0,
					epoch:      16777216,
					slotOffset: 16,
					after:      periodicEpochBaseDir + "/4096/16777216/0x0232521756a0b965eab2c2245d7ad85feaeaf5f427cd14d1a7531f9d555b415c/0.ssz",
				},
				{
					before:     periodicEpochBaseDir + "/4096/16777217/0x42eabe3d2c125410cd226de6f2825fb7575ab896c3f52e43de1fa29e4c809aba/0.ssz",
					root:       "0x42eabe3d2c125410cd226de6f2825fb7575ab896c3f52e43de1fa29e4c809aba",
					index:      0,
					epoch:      16777217,
					slotOffset: 16,
					after:      periodicEpochBaseDir + "/4096/16777217/0x42eabe3d2c125410cd226de6f2825fb7575ab896c3f52e43de1fa29e4c809aba/0.ssz",
				},
				{
					before:     "0x0232521756a0b965eab2c2245d7ad85feaeaf5f427cd14d1a7531f9d555b415c/0.ssz",
					root:       "0x0232521756a0b965eab2c2245d7ad85feaeaf5f427cd14d1a7531f9d555b415c",
					index:      0,
					epoch:      16777216,
					slotOffset: 16,
					after:      periodicEpochBaseDir + "/4096/16777216/0x0232521756a0b965eab2c2245d7ad85feaeaf5f427cd14d1a7531f9d555b415c/0.ssz",
				},
				{
					before:     periodicEpochBaseDir + "/4096/16777216/0x2326de064f828c564740da17fc247b30d7e7300da24b0aae39a0c91791acc19f/0.ssz",
					root:       "0x2326de064f828c564740da17fc247b30d7e7300da24b0aae39a0c91791acc19f",
					index:      0,
					epoch:      16777216,
					slotOffset: 31,
					after:      periodicEpochBaseDir + "/4096/16777216/0x2326de064f828c564740da17fc247b30d7e7300da24b0aae39a0c91791acc19f/0.ssz",
				},
				{
					before:     periodicEpochBaseDir + "/2/11235/0x666cea5034e22bd3b849cb33914cad59afd88ee08e4d5bc0e997411c945fbc1d/1.ssz",
					root:       "0x666cea5034e22bd3b849cb33914cad59afd88ee08e4d5bc0e997411c945fbc1d",
					index:      1,
					epoch:      11235,
					slotOffset: 0,
					after:      periodicEpochBaseDir + "/2/11235/0x666cea5034e22bd3b849cb33914cad59afd88ee08e4d5bc0e997411c945fbc1d/1.ssz",
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fs, bs := NewEphemeralBlobStorageAndFs(t)
			from := &flatRootLayout{fs: fs}
			cache := newBlobStorageCache()
			pruner := newBlobPruner(bs.retentionEpochs)
			to, err := newPeriodicEpochLayout(fs, cache, pruner)
			require.NoError(t, err)
			testSetupPaths(t, fs, c.plan)
			err = migrateLayout(fs, from, to, cache)
			if c.err != nil {
				require.ErrorIs(t, err, c.err)
				return
			}
			require.NoError(t, err)
			require.NoError(t, warmCache(bs.layout, bs.cache))
			testAssertNewPaths(t, fs, bs, c.plan)
		})
	}
}
