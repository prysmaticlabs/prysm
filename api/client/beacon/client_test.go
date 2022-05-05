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
			name: "complete version",
			v:    "Prysm/v2.0.6 (linux amd64)",
			nv: &NodeVersion{
				implementation: "Prysm",
				semver:         "v2.0.6",
				systemInfo:     "(linux amd64)",
			},
		},
		{
			name: "nimbus version",
			v:    "Nimbus/v22.4.0-039bec-stateofus",
			nv: &NodeVersion{
				implementation: "Nimbus",
				semver:         "v22.4.0-039bec-stateofus",
				systemInfo:     "",
			},
		},
		{
			name: "teku version",
			v:    "teku/v22.3.2/linux-x86_64/oracle-java-11",
			nv: &NodeVersion{
				implementation: "teku",
				semver:         "v22.3.2",
				systemInfo:     "linux-x86_64/oracle-java-11",
			},
		},
		{
			name: "lighthouse version",
			v:    "Lighthouse/v2.1.1-5f628a7/x86_64-linux",
			nv: &NodeVersion{
				implementation: "Lighthouse",
				semver:         "v2.1.1-5f628a7",
				systemInfo:     "x86_64-linux",
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
