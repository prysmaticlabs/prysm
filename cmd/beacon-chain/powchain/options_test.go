package powchaincmd

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/prysmaticlabs/prysm/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/io/file"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"github.com/urfave/cli/v2"
)

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

func Test_parseJWTSecret(t *testing.T) {
	//func parseJWTSecret(c *cli.Context) ([]byte, error) {
	//jwtSecretFile := c.String(flags.ExecutionJWTSecretFlag.Name)
	//if jwtSecretFile == "" {
	//return nil, nil
	//}
	//enc, err := file.ReadFileAsBytes(jwtSecretFile)
	//if err != nil {
	//return nil, err
	//}
	//if len(enc) == 0 {
	//return nil, fmt.Errorf("provided JWT secret in file %s cannot be empty", jwtSecretFile)
	//}
	//secret, err := hexutil.Decode(string(enc))
	//if err != nil {
	//return nil, err
	//}
	//if len(enc) != 32 {
	//return nil, errors.New("provided JWT secret should be a hex string of 32 bytes")
	//}
	//return secret, nil
	//}
	t.Run("no flag value specified leads to nil secret", func(t *testing.T) {
		app := cli.App{}
		set := flag.NewFlagSet("test", 0)
		set.String(flags.ExecutionJWTSecretFlag.Name, "", "")
		ctx := cli.NewContext(&app, set, nil)
		secret, err := parseJWTSecret(ctx)
		require.NoError(t, err)
		require.Equal(t, true, secret == nil)
	})
	t.Run("flag specified but no file found", func(t *testing.T) {
		app := cli.App{}
		set := flag.NewFlagSet("test", 0)
		set.String(flags.ExecutionJWTSecretFlag.Name, "/tmp/askdjkajsd", "")
		ctx := cli.NewContext(&app, set, nil)
		_, err := parseJWTSecret(ctx)
		require.ErrorContains(t, "no such file", err)
	})
	t.Run("empty string in file", func(t *testing.T) {
		app := cli.App{}
		set := flag.NewFlagSet("test", 0)
		fullPath := filepath.Join(os.TempDir(), "foohex")
		require.NoError(t, file.WriteFile(fullPath, []byte{}))
		t.Cleanup(func() {
			if err := os.RemoveAll(fullPath); err != nil {
				t.Fatalf("Could not delete temp dir: %v", err)
			}
		})
		set.String(flags.ExecutionJWTSecretFlag.Name, fullPath, "")
		ctx := cli.NewContext(&app, set, nil)
		_, err := parseJWTSecret(ctx)
		require.ErrorContains(t, "cannot be empty", err)
	})
	t.Run("not 32 bytes", func(t *testing.T) {
		app := cli.App{}
		set := flag.NewFlagSet("test", 0)
		fullPath := filepath.Join(os.TempDir(), "foohex")
		secret := bytesutil.PadTo([]byte("foo"), 64)
		hexData := fmt.Sprintf("%#x", secret)
		require.NoError(t, file.WriteFile(fullPath, []byte(hexData)))
		t.Cleanup(func() {
			if err := os.RemoveAll(fullPath); err != nil {
				t.Fatalf("Could not delete temp dir: %v", err)
			}
		})
		set.String(flags.ExecutionJWTSecretFlag.Name, fullPath, "")
		ctx := cli.NewContext(&app, set, nil)
		_, err := parseJWTSecret(ctx)
		require.ErrorContains(t, "should be a hex string of 32 bytes", err)
	})
	t.Run("bad data", func(t *testing.T) {
		app := cli.App{}
		set := flag.NewFlagSet("test", 0)
		fullPath := filepath.Join(os.TempDir(), "foohex")
		secret := []byte("foo")
		require.NoError(t, file.WriteFile(fullPath, secret))
		t.Cleanup(func() {
			if err := os.RemoveAll(fullPath); err != nil {
				t.Fatalf("Could not delete temp dir: %v", err)
			}
		})
		set.String(flags.ExecutionJWTSecretFlag.Name, fullPath, "")
		ctx := cli.NewContext(&app, set, nil)
		_, err := parseJWTSecret(ctx)
		require.ErrorContains(t, "invalid byte", err)
	})
	t.Run("correct format", func(t *testing.T) {
		app := cli.App{}
		set := flag.NewFlagSet("test", 0)
		fullPath := filepath.Join(os.TempDir(), "foohex")
		secret := bytesutil.ToBytes32([]byte("foo"))
		secretHex := fmt.Sprintf("%#x", secret)
		require.NoError(t, file.WriteFile(fullPath, []byte(secretHex)))
		t.Cleanup(func() {
			if err := os.RemoveAll(fullPath); err != nil {
				t.Fatalf("Could not delete temp dir: %v", err)
			}
		})
		set.String(flags.ExecutionJWTSecretFlag.Name, fullPath, "")
		ctx := cli.NewContext(&app, set, nil)
		got, err := parseJWTSecret(ctx)
		require.NoError(t, err)
		require.DeepEqual(t, secret[:], got)
	})
}

func TestPowchainPreregistration_EmptyWeb3Provider(t *testing.T) {
	hook := logTest.NewGlobal()
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String(flags.HTTPWeb3ProviderFlag.Name, "", "")
	fallback := cli.StringSlice{}
	set.Var(&fallback, flags.FallbackWeb3ProviderFlag.Name, "")
	ctx := cli.NewContext(&app, set, nil)
	parsePowchainEndpoints(ctx)
	assert.LogsContain(t, hook, "No ETH1 node specified to run with the beacon node")
}
