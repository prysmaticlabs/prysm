package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/spf13/afero"
)

func testSetupPaths(t *testing.T, fs afero.Fs, paths []migrateBeforeAfter) {
	for _, ba := range paths {
		p := ba.before
		dir := filepath.Dir(p)
		require.NoError(t, fs.MkdirAll(dir, directoryPermissions))
		fh, err := fs.Create(p)
		require.NoError(t, err)
		require.NoError(t, fh.Close())
		// double check that we got the full path correct
		_, err = fs.Stat(ba.before)
		require.NoError(t, err)
	}
}

func testAssertNewPaths(t *testing.T, fs afero.Fs, paths []migrateBeforeAfter) {
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
	}
}

type migrateBeforeAfter struct {
	before string
	after  string
}

func TestOneBytePrefixMigrator(t *testing.T) {
	cases := []struct {
		name string
		plan []migrateBeforeAfter
		err  error
	}{
		{
			name: "happy path",
			plan: []migrateBeforeAfter{
				{
					before: "0x0125e54c64c925018c9296965a5b622d9f5ab626c10917860dcfb6aa09a0a00b/0.ssz",
					after:  "0x01/0x0125e54c64c925018c9296965a5b622d9f5ab626c10917860dcfb6aa09a0a00b/0.ssz",
				},
				{
					before: "0x0127dba6fd30fdbb47e73e861d5c6e602b38ac3ddc945bb6a2fc4e10761e9a86/0.ssz",
					after:  "0x01/0x0127dba6fd30fdbb47e73e861d5c6e602b38ac3ddc945bb6a2fc4e10761e9a86/0.ssz",
				},
				{
					before: "0x0127dba6fd30fdbb47e73e861d5c6e602b38ac3ddc945bb6a2fc4e10761e9a86/1.ssz",
					after:  "0x01/0x0127dba6fd30fdbb47e73e861d5c6e602b38ac3ddc945bb6a2fc4e10761e9a86/1.ssz",
				},
				{
					before: "0x0232521756a0b965eab2c2245d7ad85feaeaf5f427cd14d1a7531f9d555b415c/0.ssz",
					after:  "0x02/0x0232521756a0b965eab2c2245d7ad85feaeaf5f427cd14d1a7531f9d555b415c/0.ssz",
				},
			},
		},
		{
			name: "different roots same prefix",
			plan: []migrateBeforeAfter{
				{
					before: "0xff/0xffff875e1d985c5ccb214894983f2428edb271f0f87b68ba7010e4a99df3b5cb/0.ssz",
					after:  "0xff/0xffff875e1d985c5ccb214894983f2428edb271f0f87b68ba7010e4a99df3b5cb/0.ssz",
				},
				{
					before: "0xff0774a80664e1667dcd5a18bced866a596b6cef5f351c0f88cd310dd00cb16d/0.ssz",
					after:  "0xff/0xff0774a80664e1667dcd5a18bced866a596b6cef5f351c0f88cd310dd00cb16d/0.ssz",
				},
				{
					before: "0x0125e54c64c925018c9296965a5b622d9f5ab626c10917860dcfb6aa09a0a00b/0.ssz",
					after:  "0x01/0x0125e54c64c925018c9296965a5b622d9f5ab626c10917860dcfb6aa09a0a00b/0.ssz",
				},
				{
					before: "0x01/0x0127dba6fd30fdbb47e73e861d5c6e602b38ac3ddc945bb6a2fc4e10761e9a86/0.ssz",
					after:  "0x01/0x0127dba6fd30fdbb47e73e861d5c6e602b38ac3ddc945bb6a2fc4e10761e9a86/0.ssz",
				},
				{
					before: "0x0127dba6fd30fdbb47e73e861d5c6e602b38ac3ddc945bb6a2fc4e10761e9a86/1.ssz",
					after:  "0x01/0x0127dba6fd30fdbb47e73e861d5c6e602b38ac3ddc945bb6a2fc4e10761e9a86/1.ssz",
				},
			},
		},
		{
			name: "mix old and new",
			plan: []migrateBeforeAfter{
				{
					before: "0xff/0xffff875e1d985c5ccb214894983f2428edb271f0f87b68ba7010e4a99df3b5cb/0.ssz",
					after:  "0xff/0xffff875e1d985c5ccb214894983f2428edb271f0f87b68ba7010e4a99df3b5cb/0.ssz",
				},
				{
					before: "0x0125e54c64c925018c9296965a5b622d9f5ab626c10917860dcfb6aa09a0a00b/0.ssz",
					after:  "0x01/0x0125e54c64c925018c9296965a5b622d9f5ab626c10917860dcfb6aa09a0a00b/0.ssz",
				},
				{
					before: "0xa0/0xa0000137a809ca8425e03ae6c4244eedc7c0aa37f2735883366bcaf1cca1e3f3/0.ssz",
					after:  "0xa0/0xa0000137a809ca8425e03ae6c4244eedc7c0aa37f2735883366bcaf1cca1e3f3/0.ssz",
				},
				{
					before: "0xa0/0xa0000137a809ca8425e03ae6c4244eedc7c0aa37f2735883366bcaf1cca1e3f3/1.ssz",
					after:  "0xa0/0xa0000137a809ca8425e03ae6c4244eedc7c0aa37f2735883366bcaf1cca1e3f3/1.ssz",
				},
				{
					before: "0x0127dba6fd30fdbb47e73e861d5c6e602b38ac3ddc945bb6a2fc4e10761e9a86/0.ssz",
					after:  "0x01/0x0127dba6fd30fdbb47e73e861d5c6e602b38ac3ddc945bb6a2fc4e10761e9a86/0.ssz",
				},
			},
		},
		{
			name: "overwrite existing root dir",
			plan: []migrateBeforeAfter{
				{
					before: "0xff/0xffff875e1d985c5ccb214894983f2428edb271f0f87b68ba7010e4a99df3b5cb/0.ssz",
					after:  "0xff/0xffff875e1d985c5ccb214894983f2428edb271f0f87b68ba7010e4a99df3b5cb/0.ssz",
				},
				{
					before: "0xffff875e1d985c5ccb214894983f2428edb271f0f87b68ba7010e4a99df3b5cb/1.ssz",
					after:  "0xff/0xffff875e1d985c5ccb214894983f2428edb271f0f87b68ba7010e4a99df3b5cb/0.ssz",
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			from := &flatRootLayout{}
			fs, bs := NewEphemeralBlobStorageWithFs(t)
			cache := newBlobStorageCache()
			pruner := newBlobPruner(bs.retentionEpochs)
			to, err := newPeriodicEpochLayout(fs, cache, pruner)
			require.NoError(t, err)
			testSetupPaths(t, fs, c.plan)
			err = migrateLayout(context.Background(), fs, from, to, cache)
			if c.err != nil {
				require.ErrorIs(t, err, c.err)
				return
			}
			require.NoError(t, err)
			testAssertNewPaths(t, fs, c.plan)
		})
	}
}
