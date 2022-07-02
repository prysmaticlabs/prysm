package node

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/prysmaticlabs/prysm/config/params"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/cmd/validator/flags"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	validatorserviceconfig "github.com/prysmaticlabs/prysm/config/validator/service"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/validator/accounts"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	remoteweb3signer "github.com/prysmaticlabs/prysm/validator/keymanager/remote-web3signer"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/urfave/cli/v2"
)

// Test that the sharding node can build with default flag values.
func TestNode_Builds(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String("datadir", t.TempDir()+"/datadir", "the node data directory")
	dir := t.TempDir() + "/walletpath"
	passwordDir := t.TempDir() + "/password"
	require.NoError(t, os.MkdirAll(passwordDir, os.ModePerm))
	passwordFile := filepath.Join(passwordDir, "password.txt")
	walletPassword := "$$Passw0rdz2$$"
	require.NoError(t, os.WriteFile(
		passwordFile,
		[]byte(walletPassword),
		os.ModePerm,
	))
	set.String("wallet-dir", dir, "path to wallet")
	set.String("wallet-password-file", passwordFile, "path to wallet password")
	set.String("keymanager-kind", "imported", "keymanager kind")
	set.String("verbosity", "debug", "log verbosity")
	require.NoError(t, set.Set(flags.WalletPasswordFileFlag.Name, passwordFile))
	ctx := cli.NewContext(&app, set, nil)
	_, err := accounts.CreateWalletWithKeymanager(ctx.Context, &accounts.CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      dir,
			KeymanagerKind: keymanager.Local,
			WalletPassword: walletPassword,
		},
	})
	require.NoError(t, err)

	valClient, err := NewValidatorClient(ctx)
	require.NoError(t, err, "Failed to create ValidatorClient")
	err = valClient.db.Close()
	require.NoError(t, err)
}

// TestClearDB tests clearing the database
func TestClearDB(t *testing.T) {
	hook := logtest.NewGlobal()
	tmp := filepath.Join(t.TempDir(), "datadirtest")
	require.NoError(t, clearDB(context.Background(), tmp, true))
	require.LogsContain(t, hook, "Removing database")
}

// TestWeb3SignerConfig tests the web3 signer config returns the correct values.
func TestWeb3SignerConfig(t *testing.T) {
	pubkey1decoded, err := hexutil.Decode("0xa99a76ed7796f7be22d5b7e85deeb7c5677e88e511e0b337618f8c4eb61349b4bf2d153f649f7b53359fe8b94a38e44c")
	require.NoError(t, err)
	bytepubkey1 := bytesutil.ToBytes48(pubkey1decoded)

	pubkey2decoded, err := hexutil.Decode("0xb89bebc699769726a318c8e9971bd3171297c61aea4a6578a7a4f94b547dcba5bac16a89108b6b6a1fe3695d1a874a0b")
	require.NoError(t, err)
	bytepubkey2 := bytesutil.ToBytes48(pubkey2decoded)

	type args struct {
		baseURL         string
		publicKeysOrURL string
	}
	tests := []struct {
		name       string
		args       args
		want       *remoteweb3signer.SetupConfig
		wantErrMsg string
	}{
		{
			name: "happy path with public keys",
			args: args{
				baseURL: "http://localhost:8545",
				publicKeysOrURL: "0xa99a76ed7796f7be22d5b7e85deeb7c5677e88e511e0b337618f8c4eb61349b4bf2d153f649f7b53359fe8b94a38e44c," +
					"0xb89bebc699769726a318c8e9971bd3171297c61aea4a6578a7a4f94b547dcba5bac16a89108b6b6a1fe3695d1a874a0b",
			},
			want: &remoteweb3signer.SetupConfig{
				BaseEndpoint:          "http://localhost:8545",
				GenesisValidatorsRoot: nil,
				PublicKeysURL:         "",
				ProvidedPublicKeys: [][48]byte{
					bytepubkey1,
					bytepubkey2,
				},
			},
		},
		{
			name: "happy path with external url",
			args: args{
				baseURL:         "http://localhost:8545",
				publicKeysOrURL: "http://localhost:8545/api/v1/eth2/publicKeys",
			},
			want: &remoteweb3signer.SetupConfig{
				BaseEndpoint:          "http://localhost:8545",
				GenesisValidatorsRoot: nil,
				PublicKeysURL:         "http://localhost:8545/api/v1/eth2/publicKeys",
				ProvidedPublicKeys:    nil,
			},
		},
		{
			name: "Bad base URL",
			args: args{
				baseURL: "0xa99a76ed7796f7be22d5b7e85deeb7c5677e88,",
				publicKeysOrURL: "0xa99a76ed7796f7be22d5b7e85deeb7c5677e88e511e0b337618f8c4eb61349b4bf2d153f649f7b53359fe8b94a38e44c," +
					"0xb89bebc699769726a318c8e9971bd3171297c61aea4a6578a7a4f94b547dcba5bac16a89108b6b6a1fe3695d1a874a0b",
			},
			want:       nil,
			wantErrMsg: "web3signer url 0xa99a76ed7796f7be22d5b7e85deeb7c5677e88, is invalid: parse \"0xa99a76ed7796f7be22d5b7e85deeb7c5677e88,\": invalid URI for request",
		},
		{
			name: "Bad publicKeys",
			args: args{
				baseURL: "http://localhost:8545",
				publicKeysOrURL: "0xa99a76ed7796f7be22c," +
					"0xb89bebc699769726a318c8e9971bd3171297c61aea4a6578a7a4f94b547dcba5bac16a89108b6b6a1fe3695d1a874a0b",
			},
			want:       nil,
			wantErrMsg: "could not decode public key for web3signer: 0xa99a76ed7796f7be22c: hex string of odd length",
		},
		{
			name: "Bad publicKeysURL",
			args: args{
				baseURL:         "http://localhost:8545",
				publicKeysOrURL: "localhost",
			},
			want:       nil,
			wantErrMsg: "could not decode public key for web3signer: localhost: hex string without 0x prefix",
		},
		{
			name: "Base URL missing scheme or host",
			args: args{
				baseURL:         "localhost:8545",
				publicKeysOrURL: "localhost",
			},
			want:       nil,
			wantErrMsg: "web3signer url must be in the format of http(s)://host:port url used: localhost:8545",
		},
		{
			name: "Public Keys URL missing scheme or host",
			args: args{
				baseURL:         "http://localhost:8545",
				publicKeysOrURL: "localhost:8545",
			},
			want:       nil,
			wantErrMsg: "could not decode public key for web3signer: localhost:8545: hex string without 0x prefix",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := web3SignerConfig(newWeb3SignerCli(t, tt.args.baseURL, tt.args.publicKeysOrURL))
			if (tt.wantErrMsg != "") && (tt.wantErrMsg != fmt.Sprintf("%v", err)) {
				t.Errorf("web3SignerConfig error = %v, wantErrMsg = %v", err, tt.wantErrMsg)
				return
			}
			require.DeepEqual(t, got, tt.want)
		})
	}
}

func newWeb3SignerCli(t *testing.T, baseUrl string, publicKeysOrURL string) *cli.Context {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String("validators-external-signer-url", baseUrl, "baseUrl")
	set.String("validators-external-signer-public-keys", publicKeysOrURL, "publicKeys or URL")
	require.NoError(t, set.Set(flags.Web3SignerURLFlag.Name, baseUrl))
	require.NoError(t, set.Set(flags.Web3SignerPublicValidatorKeysFlag.Name, publicKeysOrURL))
	return cli.NewContext(&app, set, nil)
}

func TestProposerSettings(t *testing.T) {
	hook := logtest.NewGlobal()

	type proposerSettingsFlag struct {
		dir        string
		url        string
		defaultfee string
	}

	type args struct {
		proposerSettingsFlagValues *proposerSettingsFlag
	}
	tests := []struct {
		name        string
		args        args
		want        func() *validatorserviceconfig.ProposerSettings
		urlResponse string
		wantErr     string
		wantLog     string
	}{
		{
			name: "Happy Path Config file File, bad checksum",
			args: args{
				proposerSettingsFlagValues: &proposerSettingsFlag{
					dir:        "./testdata/good-prepare-beacon-proposer-config-badchecksum.json",
					url:        "",
					defaultfee: "",
				},
			},
			want: func() *validatorserviceconfig.ProposerSettings {
				key1, err := hexutil.Decode("0xa057816155ad77931185101128655c0191bd0214c201ca48ed887f6c4c6adf334070efcd75140eada5ac83a92506dd7a")
				require.NoError(t, err)
				return &validatorserviceconfig.ProposerSettings{
					ProposeConfig: map[[fieldparams.BLSPubkeyLength]byte]*validatorserviceconfig.ProposerOption{
						bytesutil.ToBytes48(key1): {
							FeeRecipient: common.HexToAddress("0xae967917c465db8578ca9024c205720b1a3651A9"),
							GasLimit:     params.BeaconConfig().DefaultBuilderGasLimit,
						},
					},
					DefaultConfig: &validatorserviceconfig.ProposerOption{
						FeeRecipient: common.HexToAddress("0xae967917c465db8578ca9024c205720b1a3651A9"),
						GasLimit:     params.BeaconConfig().DefaultBuilderGasLimit,
					},
				}
			},
			wantErr: "",
			wantLog: "is not a checksum Ethereum address",
		},
		{
			name: "Happy Path Config file File multiple fee recipients",
			args: args{
				proposerSettingsFlagValues: &proposerSettingsFlag{
					dir:        "./testdata/good-prepare-beacon-proposer-config-multiple.json",
					url:        "",
					defaultfee: "",
				},
			},
			want: func() *validatorserviceconfig.ProposerSettings {
				key1, err := hexutil.Decode("0xa057816155ad77931185101128655c0191bd0214c201ca48ed887f6c4c6adf334070efcd75140eada5ac83a92506dd7a")
				require.NoError(t, err)
				key2, err := hexutil.Decode("0xb057816155ad77931185101128655c0191bd0214c201ca48ed887f6c4c6adf334070efcd75140eada5ac83a92506dd7b")
				require.NoError(t, err)
				return &validatorserviceconfig.ProposerSettings{
					ProposeConfig: map[[fieldparams.BLSPubkeyLength]byte]*validatorserviceconfig.ProposerOption{
						bytesutil.ToBytes48(key1): {
							FeeRecipient: common.HexToAddress("0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3"),
							GasLimit:     params.BeaconConfig().DefaultBuilderGasLimit,
						},
						bytesutil.ToBytes48(key2): {
							FeeRecipient: common.HexToAddress("0x60155530FCE8a85ec7055A5F8b2bE214B3DaeFd4"),
							GasLimit:     params.BeaconConfig().DefaultBuilderGasLimit,
						},
					},
					DefaultConfig: &validatorserviceconfig.ProposerOption{
						FeeRecipient: common.HexToAddress("0x6e35733c5af9B61374A128e6F85f553aF09ff89A"),
						GasLimit:     params.BeaconConfig().DefaultBuilderGasLimit,
					},
				}
			},
			wantErr: "",
		},
		{
			name: "Happy Path Config URL File",
			args: args{
				proposerSettingsFlagValues: &proposerSettingsFlag{
					dir:        "",
					url:        "./testdata/good-prepare-beacon-proposer-config.json",
					defaultfee: "",
				},
			},
			want: func() *validatorserviceconfig.ProposerSettings {
				key1, err := hexutil.Decode("0xa057816155ad77931185101128655c0191bd0214c201ca48ed887f6c4c6adf334070efcd75140eada5ac83a92506dd7a")
				require.NoError(t, err)
				return &validatorserviceconfig.ProposerSettings{
					ProposeConfig: map[[fieldparams.BLSPubkeyLength]byte]*validatorserviceconfig.ProposerOption{
						bytesutil.ToBytes48(key1): {
							FeeRecipient: common.HexToAddress("0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3"),
							GasLimit:     params.BeaconConfig().DefaultBuilderGasLimit,
						},
					},
					DefaultConfig: &validatorserviceconfig.ProposerOption{
						FeeRecipient: common.HexToAddress("0x6e35733c5af9B61374A128e6F85f553aF09ff89A"),
						GasLimit:     params.BeaconConfig().DefaultBuilderGasLimit,
					},
				}
			},
			wantErr: "",
		},
		{
			name: "Happy Path Config YAML file with custom Gas Limit",
			args: args{
				proposerSettingsFlagValues: &proposerSettingsFlag{
					dir:        "./testdata/good-prepare-beacon-proposer-config.yaml",
					url:        "",
					defaultfee: "",
				},
			},
			want: func() *validatorserviceconfig.ProposerSettings {
				key1, err := hexutil.Decode("0xa057816155ad77931185101128655c0191bd0214c201ca48ed887f6c4c6adf334070efcd75140eada5ac83a92506dd7a")
				require.NoError(t, err)
				return &validatorserviceconfig.ProposerSettings{
					ProposeConfig: map[[fieldparams.BLSPubkeyLength]byte]*validatorserviceconfig.ProposerOption{
						bytesutil.ToBytes48(key1): {
							FeeRecipient: common.HexToAddress("0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3"),
							GasLimit:     uint64(40000000),
						},
					},
					DefaultConfig: &validatorserviceconfig.ProposerOption{
						FeeRecipient: common.HexToAddress("0x6e35733c5af9B61374A128e6F85f553aF09ff89A"),
						GasLimit:     uint64(45000000),
					},
				}
			},
			wantErr: "",
		},
		{
			name: "Happy Path Suggested Fee File",
			args: args{
				proposerSettingsFlagValues: &proposerSettingsFlag{
					dir:        "",
					url:        "",
					defaultfee: "0x6e35733c5af9B61374A128e6F85f553aF09ff89A",
				},
			},
			want: func() *validatorserviceconfig.ProposerSettings {
				return &validatorserviceconfig.ProposerSettings{
					ProposeConfig: nil,
					DefaultConfig: &validatorserviceconfig.ProposerOption{
						FeeRecipient: common.HexToAddress("0x6e35733c5af9B61374A128e6F85f553aF09ff89A"),
						GasLimit:     params.BeaconConfig().DefaultBuilderGasLimit,
					},
				}
			},
			wantErr: "",
		},
		{
			name: "Suggested Fee does not Override Config",
			args: args{
				proposerSettingsFlagValues: &proposerSettingsFlag{
					dir:        "./testdata/good-prepare-beacon-proposer-config.json",
					url:        "",
					defaultfee: "0x6e35733c5af9B61374A128e6F85f553aF09ff89B",
				},
			},
			want: func() *validatorserviceconfig.ProposerSettings {
				key1, err := hexutil.Decode("0xa057816155ad77931185101128655c0191bd0214c201ca48ed887f6c4c6adf334070efcd75140eada5ac83a92506dd7a")
				require.NoError(t, err)
				return &validatorserviceconfig.ProposerSettings{
					ProposeConfig: map[[fieldparams.BLSPubkeyLength]byte]*validatorserviceconfig.ProposerOption{
						bytesutil.ToBytes48(key1): {
							FeeRecipient: common.HexToAddress("0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3"),
							GasLimit:     params.BeaconConfig().DefaultBuilderGasLimit,
						},
					},
					DefaultConfig: &validatorserviceconfig.ProposerOption{
						FeeRecipient: common.HexToAddress("0x6e35733c5af9B61374A128e6F85f553aF09ff89A"),
						GasLimit:     params.BeaconConfig().DefaultBuilderGasLimit,
					},
				}
			},
			wantErr: "",
		},
		{
			name: "No flags set means empty config",
			args: args{
				proposerSettingsFlagValues: &proposerSettingsFlag{
					dir:        "",
					url:        "",
					defaultfee: "",
				},
			},
			want: func() *validatorserviceconfig.ProposerSettings {
				return nil
			},
			wantErr: "",
		},
		{
			name: "Bad File Path",
			args: args{
				proposerSettingsFlagValues: &proposerSettingsFlag{
					dir:        "./testdata/bad-prepare-beacon-proposer-config.json",
					url:        "",
					defaultfee: "",
				},
			},
			want: func() *validatorserviceconfig.ProposerSettings {
				return nil
			},
			wantErr: "failed to unmarshal yaml file",
		},
		{
			name: "Both URL and Dir flags used resulting in error",
			args: args{
				proposerSettingsFlagValues: &proposerSettingsFlag{
					dir:        "./testdata/good-prepare-beacon-proposer-config.json",
					url:        "./testdata/good-prepare-beacon-proposer-config.json",
					defaultfee: "",
				},
			},
			want: func() *validatorserviceconfig.ProposerSettings {
				return &validatorserviceconfig.ProposerSettings{}
			},
			wantErr: "cannot specify both",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := cli.App{}
			set := flag.NewFlagSet("test", 0)
			if tt.args.proposerSettingsFlagValues.dir != "" {
				set.String(flags.ProposerSettingsFlag.Name, tt.args.proposerSettingsFlagValues.dir, "")
				require.NoError(t, set.Set(flags.ProposerSettingsFlag.Name, tt.args.proposerSettingsFlagValues.dir))
			}
			if tt.args.proposerSettingsFlagValues.url != "" {
				content, err := os.ReadFile(tt.args.proposerSettingsFlagValues.url)
				require.NoError(t, err)
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(200)
					w.Header().Set("Content-Type", "application/json")
					_, err := fmt.Fprintf(w, string(content))
					require.NoError(t, err)
				}))
				defer srv.Close()

				set.String(flags.ProposerSettingsURLFlag.Name, tt.args.proposerSettingsFlagValues.url, "")
				require.NoError(t, set.Set(flags.ProposerSettingsURLFlag.Name, srv.URL))
			}
			if tt.args.proposerSettingsFlagValues.defaultfee != "" {
				set.String(flags.SuggestedFeeRecipientFlag.Name, tt.args.proposerSettingsFlagValues.defaultfee, "")
				require.NoError(t, set.Set(flags.SuggestedFeeRecipientFlag.Name, tt.args.proposerSettingsFlagValues.defaultfee))
			}
			cliCtx := cli.NewContext(&app, set, nil)
			got, err := proposerSettings(cliCtx)
			if tt.wantErr != "" {
				require.ErrorContains(t, tt.wantErr, err)
				return
			}
			if tt.wantLog != "" {
				assert.LogsContain(t, hook,
					tt.wantLog,
				)
			}
			w := tt.want()
			require.DeepEqual(t, w, got)
		})
	}
}
