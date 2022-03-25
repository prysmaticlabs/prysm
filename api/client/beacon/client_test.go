package beacon

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestParseNodeVersion(t *testing.T) {
	cases := []struct {
		name string
		v    string
		err  error
		nv   *NodeVersion
	}{
		{
			name: "empty string",
			v:    "",
			err:  ErrInvalidNodeVersion,
		},
		{
			name: "Prysm as the version string",
			v:    "Prysm",
			err:  ErrInvalidNodeVersion,
		},
		{
			name: "semver only",
			v:    "v2.0.6",
			err:  ErrInvalidNodeVersion,
		},
		{
			name: "implementation and semver only",
			v:    "Prysm/v2.0.6",
			err:  ErrInvalidNodeVersion,
		},
		{
			name: "complete version",
			v:    "Prysm/v2.0.6 (linux amd64)",
			nv: &NodeVersion{
				implementation: "Prysm",
				semver:         "v2.0.6",
				systemInfo:     "linux amd64",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			nv, err := parseNodeVersion(c.v)
			if c.err != nil {
				require.ErrorIs(t, err, c.err)
			} else {
				require.NoError(t, err)
				require.DeepEqual(t, c.nv, nv)
			}
		})
	}
}
