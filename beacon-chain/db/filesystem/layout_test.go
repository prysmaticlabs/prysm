package filesystem

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v5/testing/require"
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
