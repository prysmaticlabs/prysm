package filesystem

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"path"
	"sort"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/verification"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
	"github.com/spf13/afero"
)

func TestRootFromDir(t *testing.T) {
	cases := []struct {
		name string
		dir  string
		err  error
		root [32]byte
	}{
		{
			name: "happy path",
			dir:  "0xffff875e1d985c5ccb214894983f2428edb271f0f87b68ba7010e4a99df3b5cb",
			root: [32]byte{255, 255, 135, 94, 29, 152, 92, 92, 203, 33, 72, 148, 152, 63, 36, 40,
				237, 178, 113, 240, 248, 123, 104, 186, 112, 16, 228, 169, 157, 243, 181, 203},
		},
		{
			name: "too short",
			dir:  "0xffff875e1d985c5ccb214894983f2428edb271f0f87b68ba7010e4a99df3b5c",
			err:  errInvalidRootString,
		},
		{
			name: "too log",
			dir:  "0xffff875e1d985c5ccb214894983f2428edb271f0f87b68ba7010e4a99df3b5cbb",
			err:  errInvalidRootString,
		},
		{
			name: "missing prefix",
			dir:  "ffff875e1d985c5ccb214894983f2428edb271f0f87b68ba7010e4a99df3b5cb",
			err:  errInvalidRootString,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root, err := stringToRoot(c.dir)
			if c.err != nil {
				require.ErrorIs(t, err, c.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, c.root, root)
		})
	}
}

func TestSlotFromFile(t *testing.T) {
	cases := []struct {
		slot primitives.Slot
	}{
		{slot: 0},
		{slot: 2},
		{slot: 1123581321},
		{slot: math.MaxUint64},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("slot %d", c.slot), func(t *testing.T) {
			fs, bs := NewEphemeralBlobStorageAndFs(t)
			_, sidecars := util.GenerateTestDenebBlockWithSidecar(t, [32]byte{}, c.slot, 1)
			sc, err := verification.BlobSidecarNoop(sidecars[0])
			require.NoError(t, err)
			require.NoError(t, bs.Save(sc))
			namer := identForSidecar(sc)
			sszPath := bs.layout.sszPath(namer)
			slot, err := slotFromFile(sszPath, fs)
			require.NoError(t, err)
			require.Equal(t, c.slot, slot)
		})
	}
}

type dirFiles struct {
	name     string
	isDir    bool
	children []dirFiles
}

func (df dirFiles) reify(t *testing.T, fs afero.Fs, base string) {
	fullPath := path.Join(base, df.name)
	if df.isDir {
		if df.name != "" {
			require.NoError(t, fs.Mkdir(fullPath, directoryPermissions))
		}
		for _, c := range df.children {
			c.reify(t, fs, fullPath)
		}
	} else {
		fp, err := fs.Create(fullPath)
		require.NoError(t, err)
		_, err = fp.WriteString("derp")
		require.NoError(t, err)
	}
}

func (df dirFiles) childNames() []string {
	cn := make([]string, len(df.children))
	for i := range df.children {
		cn[i] = df.children[i].name
	}
	return cn
}

func TestListDir(t *testing.T) {
	fs := afero.NewMemMapFs()
	rootStrs := []string{
		"0x0023dc5d063c7c1b37016bb54963c6ff4bfe5dfdf6dac29e7ceeb2b8fa81ed7a",
		"0xff30526cd634a5af3a09cc9bff67f33a621fc5b975750bb4432f74df077554b4",
		"0x23f5f795aaeb78c01fadaf3d06da2e99bd4b3622ae4dfea61b05b7d9adb119c2",
	}

	// parent directory
	tree := dirFiles{isDir: true}
	// break out each subdir for easier assertions
	notABlob := dirFiles{name: "notABlob", isDir: true}
	childlessBlob := dirFiles{name: rootStrs[0], isDir: true}
	blobWithSsz := dirFiles{name: rootStrs[1], isDir: true,
		children: []dirFiles{{name: "1.ssz"}, {name: "2.ssz"}},
	}
	blobWithSszAndTmp := dirFiles{name: rootStrs[2], isDir: true,
		children: []dirFiles{{name: "5.ssz"}, {name: "0.part"}}}
	tree.children = append(tree.children,
		notABlob, childlessBlob, blobWithSsz, blobWithSszAndTmp)

	topChildren := make([]string, len(tree.children))
	for i := range tree.children {
		topChildren[i] = tree.children[i].name
	}

	var filter = func(entries []string, filt func(string) bool) []string {
		filtered := make([]string, 0, len(entries))
		for i := range entries {
			if filt(entries[i]) {
				filtered = append(filtered, entries[i])
			}
		}
		return filtered
	}

	tree.reify(t, fs, "")
	cases := []struct {
		name     string
		dirPath  string
		expected []string
		filter   func(string) bool
		err      error
	}{
		{
			name:     "non-existent",
			dirPath:  "derp",
			expected: []string{},
			err:      os.ErrNotExist,
		},
		{
			name:     "empty",
			dirPath:  childlessBlob.name,
			expected: []string{},
		},
		{
			name:     "top",
			dirPath:  ".",
			expected: topChildren,
		},
		{
			name:     "custom filter: only notABlob",
			dirPath:  ".",
			expected: []string{notABlob.name},
			filter: func(s string) bool {
				return s == notABlob.name
			},
		},
		{
			name:     "root filter",
			dirPath:  ".",
			expected: []string{childlessBlob.name, blobWithSsz.name, blobWithSszAndTmp.name},
			filter:   isRootDir,
		},
		{
			name:     "ssz filter",
			dirPath:  blobWithSsz.name,
			expected: blobWithSsz.childNames(),
			filter:   isSszFile,
		},
		{
			name:     "ssz mixed filter",
			dirPath:  blobWithSszAndTmp.name,
			expected: []string{"5.ssz"},
			filter:   isSszFile,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result, err := listDir(fs, c.dirPath)
			if c.filter != nil {
				result = filter(result, c.filter)
			}
			if c.err != nil {
				require.ErrorIs(t, err, c.err)
				require.Equal(t, 0, len(result))
			} else {
				require.NoError(t, err)
				sort.Strings(c.expected)
				sort.Strings(result)
				require.DeepEqual(t, c.expected, result)
			}
		})
	}
}

func TestSlotFromBlob(t *testing.T) {
	cases := []struct {
		slot primitives.Slot
	}{
		{slot: 0},
		{slot: 2},
		{slot: 1123581321},
		{slot: math.MaxUint64},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("slot %d", c.slot), func(t *testing.T) {
			_, sidecars := util.GenerateTestDenebBlockWithSidecar(t, [32]byte{}, c.slot, 1)
			sc := sidecars[0]
			enc, err := sc.MarshalSSZ()
			require.NoError(t, err)
			slot, err := slotFromBlob(bytes.NewReader(enc))
			require.NoError(t, err)
			require.Equal(t, c.slot, slot)
		})
	}
}
