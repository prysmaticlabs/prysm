package node

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/cmd/validator/flags"
	validator_service_config "github.com/prysmaticlabs/prysm/config/validator/service"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/validator/accounts"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	remote_web3signer "github.com/prysmaticlabs/prysm/validator/keymanager/remote-web3signer"
	logTest "github.com/sirupsen/logrus/hooks/test"
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
	require.NoError(t, ioutil.WriteFile(
		passwordFile,
		[]byte(walletPassword),
		os.ModePerm,
	))
	set.String("wallet-dir", dir, "path to wallet")
	set.String("wallet-password-file", passwordFile, "path to wallet password")
	set.String("keymanager-kind", "imported", "keymanager kind")
	set.String("verbosity", "debug", "log verbosity")
	require.NoError(t, set.Set(flags.WalletPasswordFileFlag.Name, passwordFile))
	context := cli.NewContext(&app, set, nil)
	_, err := accounts.CreateWalletWithKeymanager(context.Context, &accounts.CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      dir,
			KeymanagerKind: keymanager.Local,
			WalletPassword: walletPassword,
		},
	})
	require.NoError(t, err)

	valClient, err := NewValidatorClient(context)
	require.NoError(t, err, "Failed to create ValidatorClient")
	err = valClient.db.Close()
	require.NoError(t, err)
}

// TestClearDB tests clearing the database
func TestClearDB(t *testing.T) {
	hook := logTest.NewGlobal()
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
		want       *remote_web3signer.SetupConfig
		wantErrMsg string
	}{
		{
			name: "happy path with public keys",
			args: args{
				baseURL: "http://localhost:8545",
				publicKeysOrURL: "0xa99a76ed7796f7be22d5b7e85deeb7c5677e88e511e0b337618f8c4eb61349b4bf2d153f649f7b53359fe8b94a38e44c," +
					"0xb89bebc699769726a318c8e9971bd3171297c61aea4a6578a7a4f94b547dcba5bac16a89108b6b6a1fe3695d1a874a0b",
			},
			want: &remote_web3signer.SetupConfig{
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
			want: &remote_web3signer.SetupConfig{
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

type test struct {
	Foo string `json:"foo"`
	Bar int    `json:"bar"`
}

func TestUnmarshalFromFile(t *testing.T) {
	ctx := context.Background()
	type args struct {
		File string
		To   interface{}
	}
	tests := []struct {
		name        string
		args        args
		want        interface{}
		urlResponse string
		wantErr     bool
	}{
		{
			name: "Happy Path File",
			args: args{
				File: "./testdata/test-unmarshal-good.json",
				To:   &test{},
			},
			want: &test{
				Foo: "foo",
				Bar: 1,
			},
			wantErr: false,
		},
		{
			name: "Bad File Path, not json",
			args: args{
				File: "./jsontools.go",
				To:   &test{},
			},
			want:    &test{},
			wantErr: true,
		},
		{
			name: "Bad File Path",
			args: args{
				File: "./testdata/test-unmarshal-bad.json",
				To:   &test{},
			},
			want:    &test{},
			wantErr: true,
		},
		{
			name: "Bad File Path, not found",
			args: args{
				File: "./test-notfound.json",
				To:   &test{},
			},
			want:    &test{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := unmarshalFromFile(ctx, tt.args.File, tt.args.To); (err != nil) != tt.wantErr {
				t.Errorf(" error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.DeepEqual(t, tt.want, tt.args.To)
		})
	}
}

func TestUnmarshalFromURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		_, err := fmt.Fprintf(w, `{ "foo": "foo", "bar": 1}`)
		require.NoError(t, err)
	}))
	defer srv.Close()
	ctx := context.Background()
	type args struct {
		URL string
		To  interface{}
	}
	tests := []struct {
		name        string
		args        args
		want        interface{}
		urlResponse string
		wantErr     bool
	}{
		{
			name: "Happy Path URL",
			args: args{
				URL: srv.URL,
				To:  &test{},
			},
			want: &test{
				Foo: "foo",
				Bar: 1,
			},
			wantErr: false,
		},
		{
			name: "Bad URL",
			args: args{
				URL: "sadjflksdjflksadjflkdj",
				To:  &test{},
			},
			want:    &test{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := unmarshalFromURL(ctx, tt.args.URL, tt.args.To); (err != nil) != tt.wantErr {
				t.Errorf(" error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.DeepEqual(t, tt.want, tt.args.To)
		})
	}
}

func TestPrepareBeaconProposalConfig(t *testing.T) {
	type proposalFlag struct {
		dir        string
		url        string
		defaultfee string
	}
	type args struct {
		proposalFlagValues *proposalFlag
	}
	tests := []struct {
		name        string
		args        args
		want        *validator_service_config.PrepareBeaconProposalFileConfig
		urlResponse string
		wantErr     string
	}{
		{
			name: "Happy Path Config file File",
			args: args{
				proposalFlagValues: &proposalFlag{
					dir:        "./testdata/good-prepare-beacon-proposer-config.json",
					url:        "",
					defaultfee: "",
				},
			},
			want: &validator_service_config.PrepareBeaconProposalFileConfig{
				ProposeConfig: map[string]*validator_service_config.ValidatorProposerOptions{
					"0xa057816155ad77931185101128655c0191bd0214c201ca48ed887f6c4c6adf334070efcd75140eada5ac83a92506dd7a": &validator_service_config.ValidatorProposerOptions{
						FeeRecipient: "0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3",
					},
				},
				DefaultConfig: &validator_service_config.ValidatorProposerOptions{
					FeeRecipient: "0x6e35733c5af9B61374A128e6F85f553aF09ff89A",
				},
			},
			wantErr: "",
		},
		{
			name: "Happy Path Config URL File",
			args: args{
				proposalFlagValues: &proposalFlag{
					dir:        "",
					url:        "./testdata/good-prepare-beacon-proposer-config.json",
					defaultfee: "",
				},
			},
			want: &validator_service_config.PrepareBeaconProposalFileConfig{
				ProposeConfig: map[string]*validator_service_config.ValidatorProposerOptions{
					"0xa057816155ad77931185101128655c0191bd0214c201ca48ed887f6c4c6adf334070efcd75140eada5ac83a92506dd7a": &validator_service_config.ValidatorProposerOptions{
						FeeRecipient: "0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3",
					},
				},
				DefaultConfig: &validator_service_config.ValidatorProposerOptions{
					FeeRecipient: "0x6e35733c5af9B61374A128e6F85f553aF09ff89A",
				},
			},
			wantErr: "",
		},
		{
			name: "Happy Path Suggested Fee File",
			args: args{
				proposalFlagValues: &proposalFlag{
					dir:        "",
					url:        "",
					defaultfee: "0x6e35733c5af9B61374A128e6F85f553aF09ff89A",
				},
			},
			want: &validator_service_config.PrepareBeaconProposalFileConfig{
				ProposeConfig: nil,
				DefaultConfig: &validator_service_config.ValidatorProposerOptions{
					FeeRecipient: "0x6e35733c5af9B61374A128e6F85f553aF09ff89A",
				},
			},
			wantErr: "",
		},
		{
			name: "Suggested Fee Override Config",
			args: args{
				proposalFlagValues: &proposalFlag{
					dir:        "./testdata/good-prepare-beacon-proposer-config.json",
					url:        "",
					defaultfee: "0x6e35733c5af9B61374A128e6F85f553aF09ff89B",
				},
			},
			want: &validator_service_config.PrepareBeaconProposalFileConfig{
				ProposeConfig: nil,
				DefaultConfig: &validator_service_config.ValidatorProposerOptions{
					FeeRecipient: "0x6e35733c5af9B61374A128e6F85f553aF09ff89B",
				},
			},
			wantErr: "",
		},
		{
			name: "Default Fee Recipient",
			args: args{
				proposalFlagValues: &proposalFlag{
					dir:        "",
					url:        "",
					defaultfee: "",
				},
			},
			want: &validator_service_config.PrepareBeaconProposalFileConfig{
				ProposeConfig: nil,
				DefaultConfig: &validator_service_config.ValidatorProposerOptions{
					FeeRecipient: "0x0000000000000000000000000000000000000000",
				},
			},
			wantErr: "",
		},
		{
			name: "Both URL and Dir flags used resulting in error",
			args: args{
				proposalFlagValues: &proposalFlag{
					dir:        "./testdata/good-prepare-beacon-proposer-config.json",
					url:        "./testdata/good-prepare-beacon-proposer-config.json",
					defaultfee: "",
				},
			},
			want:    &validator_service_config.PrepareBeaconProposalFileConfig{},
			wantErr: "cannot specify both",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := cli.App{}
			set := flag.NewFlagSet("test", 0)
			if tt.args.proposalFlagValues.dir != "" {
				set.String("validators-proposer-config-dir", tt.args.proposalFlagValues.dir, "")
				require.NoError(t, set.Set(flags.ValidatorsProposerConfigDirFlag.Name, tt.args.proposalFlagValues.dir))
			}
			if tt.args.proposalFlagValues.url != "" {
				content, err := ioutil.ReadFile(tt.args.proposalFlagValues.url)
				require.NoError(t, err)
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(200)
					w.Header().Set("Content-Type", "application/json")
					_, err := fmt.Fprintf(w, string(content))
					require.NoError(t, err)
				}))
				defer srv.Close()

				set.String("validators-proposer-config-url", tt.args.proposalFlagValues.url, "")
				require.NoError(t, set.Set(flags.ValidatorsProposerConfigURLFlag.Name, srv.URL))
			}
			if tt.args.proposalFlagValues.defaultfee != "" {
				set.String("suggested-fee-recipient", tt.args.proposalFlagValues.defaultfee, "")
				require.NoError(t, set.Set(flags.SuggestedFeeRecipientFlag.Name, tt.args.proposalFlagValues.defaultfee))
			}
			cliCtx := cli.NewContext(&app, set, nil)
			got, err := prepareBeaconProposalConfig(cliCtx)
			if tt.wantErr != "" {
				require.ErrorContains(t, tt.wantErr, err)
				return
			}
			require.DeepEqual(t, tt.want, got)
		})
	}
}
