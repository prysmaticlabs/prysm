package jwt

import (
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/io/file"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/urfave/cli/v2"
)

// ensure length is what we expect: https://github.com/ethereum/execution-apis/issues/162

// DEBT: Copied from options_test.go
func TestPowchainCmd(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String(flags.HTTPWeb3ProviderFlag.Name, "primary", "")
	fallback := cli.StringSlice{}
	err := fallback.Set("fallback1")
	require.NoError(t, err)
	err = fallback.Set("fallback2")
	require.NoError(t, err)
	set.Var(&fallback, flags.FallbackWeb3ProviderFlag.Name, "")
	ctx := cli.NewContext(&app, set, nil)

	endpoints := parsePowchainEndpoints(ctx)
	assert.DeepEqual(t, []string{"primary", "fallback1", "fallback2"}, endpoints)
}

// DEBT: Copied from options_test.go
func Test_parseJWTSecretFromFile(t *testing.T) {
	t.Run("no flag value specified leads to nil secret", func(t *testing.T) {
		app := cli.App{}
		set := flag.NewFlagSet("test", 0)
		set.String(flags.ExecutionJWTSecretFlag.Name, "", "")
		ctx := cli.NewContext(&app, set, nil)
		secret, err := parseJWTSecretFromFile(ctx)
		require.NoError(t, err)
		require.Equal(t, true, secret == nil)
	})
	t.Run("flag specified but no file found", func(t *testing.T) {
		app := cli.App{}
		set := flag.NewFlagSet("test", 0)
		set.String(flags.ExecutionJWTSecretFlag.Name, "/tmp/askdjkajsd", "")
		ctx := cli.NewContext(&app, set, nil)
		_, err := parseJWTSecretFromFile(ctx)
		require.ErrorContains(t, "no such file", err)
	})
	t.Run("empty string in file", func(t *testing.T) {
		app := cli.App{}
		set := flag.NewFlagSet("test", 0)
		fullPath := filepath.Join(t.TempDir(), "foohex")
		require.NoError(t, file.WriteFile(fullPath, []byte{}))
		set.String(flags.ExecutionJWTSecretFlag.Name, fullPath, "")
		ctx := cli.NewContext(&app, set, nil)
		_, err := parseJWTSecretFromFile(ctx)
		require.ErrorContains(t, "cannot be empty", err)
	})
	t.Run("less than 32 bytes", func(t *testing.T) {
		app := cli.App{}
		set := flag.NewFlagSet("test", 0)
		fullPath := filepath.Join(t.TempDir(), "foohex")
		secret := bytesutil.PadTo([]byte("foo"), 31)
		hexData := fmt.Sprintf("%#x", secret)
		require.NoError(t, file.WriteFile(fullPath, []byte(hexData)))
		set.String(flags.ExecutionJWTSecretFlag.Name, fullPath, "")
		ctx := cli.NewContext(&app, set, nil)
		_, err := parseJWTSecretFromFile(ctx)
		require.ErrorContains(t, "should be a hex string of at least 32 bytes", err)
	})
	t.Run("bad data", func(t *testing.T) {
		app := cli.App{}
		set := flag.NewFlagSet("test", 0)
		fullPath := filepath.Join(t.TempDir(), "foohex")
		secret := []byte("foo")
		require.NoError(t, file.WriteFile(fullPath, secret))
		set.String(flags.ExecutionJWTSecretFlag.Name, fullPath, "")
		ctx := cli.NewContext(&app, set, nil)
		_, err := parseJWTSecretFromFile(ctx)
		require.ErrorContains(t, "invalid byte", err)
	})
	t.Run("correct format", func(t *testing.T) {
		app := cli.App{}
		set := flag.NewFlagSet("test", 0)
		fullPath := filepath.Join(t.TempDir(), "foohex")
		secret := bytesutil.ToBytes32([]byte("foo"))
		secretHex := fmt.Sprintf("%#x", secret)
		require.NoError(t, file.WriteFile(fullPath, []byte(secretHex)))
		set.String(flags.ExecutionJWTSecretFlag.Name, fullPath, "")
		ctx := cli.NewContext(&app, set, nil)
		got, err := parseJWTSecretFromFile(ctx)
		require.NoError(t, err)
		require.DeepEqual(t, secret[:], got)
	})
}

// DEBT: Copied from options.go
func parseJWTSecretFromFile(c *cli.Context) ([]byte, error) {
	jwtSecretFile := c.String(flags.ExecutionJWTSecretFlag.Name)
	if jwtSecretFile == "" {
		return nil, nil
	}
	enc, err := file.ReadFileAsBytes(jwtSecretFile)
	if err != nil {
		return nil, err
	}
	strData := strings.TrimSpace(string(enc))
	if len(strData) == 0 {
		return nil, fmt.Errorf("provided JWT secret in file %s cannot be empty", jwtSecretFile)
	}
	secret, err := hex.DecodeString(strings.TrimPrefix(strData, "0x"))
	if err != nil {
		return nil, err
	}
	if len(secret) < 32 {
		return nil, errors.New("provided JWT secret should be a hex string of at least 32 bytes")
	}
	return secret, nil
}
