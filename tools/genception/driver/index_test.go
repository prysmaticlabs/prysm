package driver

import (
	"fmt"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestJsonList(t *testing.T) {
	path := "testdata/json-list.json"
	files, err := ReadJsonIndex(path)
	require.NoError(t, err)
	require.Equal(t, 4, len(files))
}

func TestJsonIndexPathFromEnv(t *testing.T) {
	cases := []struct {
		val     string
		err     error
		envname string
		getter  func() (string, error)
	}{
		{
			getter: JsonIndexPathFromEnv,
			err:    ErrUnsetEnvVar,
		},
		{
			getter:  JsonIndexPathFromEnv,
			envname: ENV_JSON_INDEX_PATH,
			val:     "/path/to/file",
		},
		{
			getter: PackagesBaseFromEnv,
			err:    ErrUnsetEnvVar,
		},
		{
			getter:  PackagesBaseFromEnv,
			envname: ENV_PACKAGES_BASE,
			val:     "/path/to/base",
		},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			if c.envname != "" {
				t.Setenv(c.envname, c.val)
			}
			v, err := c.getter()
			if c.err != nil {
				require.ErrorIs(t, err, c.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, c.val, v)
		})
	}
}
