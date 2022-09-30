package accounts

import (
	"flag"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/v3/cmd/validator/flags"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/validator/keymanager"
	"github.com/urfave/cli/v2"
)

func TestWalletWithKeymanager_web3signer(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String(flags.Web3SignerURLFlag.Name, "http://localhost:12345", "web3signer")
	c := &cli.StringSliceFlag{
		Name: "validators-external-signer-public-keys",
	}
	err := c.Apply(set)
	require.NoError(t, err)
	require.NoError(t, set.Set(flags.Web3SignerURLFlag.Name, "http://localhost:12345"))
	require.NoError(t, set.Set(flags.Web3SignerPublicValidatorKeysFlag.Name, "0xa2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820"))
	ctx := cli.NewContext(&app, set, nil)
	bytes, err := hexutil.Decode("0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2")
	require.NoError(t, err)
	w, k, err := walletWithKeymanager(ctx, bytes)
	require.NoError(t, err)
	keys, err := k.FetchValidatingPublicKeys(ctx.Context)
	require.NoError(t, err)
	require.Equal(t, len(keys), 1)
	require.Equal(t, w.KeymanagerKind(), keymanager.Web3Signer)
}
