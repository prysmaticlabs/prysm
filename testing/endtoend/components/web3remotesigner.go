package components

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	"github.com/prysmaticlabs/prysm/runtime/interop"
	e2e "github.com/prysmaticlabs/prysm/testing/endtoend/params"
	e2etypes "github.com/prysmaticlabs/prysm/testing/endtoend/types"
)

const Web3RemoteSignerPort = 9000

var _ e2etypes.ComponentRunner = (*Web3RemoteSigner)(nil)

// rawKeyFile used for consensys's web3signer config files.
// See: https://docs.web3signer.consensys.net/en/latest/Reference/Key-Configuration-Files/#raw-unencrypted-files
type rawKeyFile struct {
	Type       string `json:"type"`       // always "file-raw" for this test.
	KeyType    string `json:"keyType"`    // always "BLS" for this test.
	PrivateKey string `json:"privateKey"` // hex encoded private key with 0x prefix.
}

type Web3RemoteSigner struct {
	ctx     context.Context
	started chan struct{}
}

func NewWeb3RemoteSigner() *Web3RemoteSigner {
	return &Web3RemoteSigner{
		started: make(chan struct{}, 1),
	}
}

// Start the web3remotesigner component with a keystore populated with the deterministic validator
// keys.
func (w *Web3RemoteSigner) Start(ctx context.Context) error {
	w.ctx = ctx

	binaryPath, found := bazel.FindBinary("", "web3signer")
	if !found {
		return errors.New("web3signer binary not found")
	}

	keystorePath := path.Join(bazel.TestTmpDir(), "web3signerkeystore")
	if err := writeKeystoreKeys(ctx, keystorePath, params.BeaconConfig().MinGenesisActiveValidatorCount); err != nil {
		return err
	}
	websignerDataDir := path.Join(bazel.TestTmpDir(), "web3signerdata")
	if err := os.MkdirAll(websignerDataDir, 0750); err != nil {
		return err
	}

	args := []string{
		// Global flags
		fmt.Sprintf("--key-store-path=%s", keystorePath),
		fmt.Sprintf("--data-path=%s", websignerDataDir),
		fmt.Sprintf("--http-listen-port=%d", Web3RemoteSignerPort),
		// Command
		"eth2",
		// Command flags
		"--network=minimal",
		"--slashing-protection-enabled=false", // Otherwise, a postgres DB is required.
		"--enable-key-manager-api=true",
	}

	cmd := exec.CommandContext(ctx, binaryPath, args...) // #nosec G204 -- Test code is safe to do this.

	// Write stdout and stderr to log files.
	stdout, err := os.Create(path.Join(e2e.TestParams.LogPath, "web3signer.stdout.log"))
	if err != nil {
		return err
	}
	stderr, err := os.Create(path.Join(e2e.TestParams.LogPath, "web3signer.stderr.log"))
	if err != nil {
		return err
	}
	defer func() {
		if err := stdout.Close(); err != nil {
			log.WithError(err).Error("Failed to close stdout file")
		}
		if err := stderr.Close(); err != nil {
			log.WithError(err).Error("Failed to close stderr file")
		}
	}()
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	log.Infof("Starting web3signer with flags: %s %s", binaryPath, strings.Join(args, " "))
	if err = cmd.Start(); err != nil {
		return err
	}

	go w.monitorStart()

	return cmd.Wait()
}

func (w *Web3RemoteSigner) Started() <-chan struct{} {
	return w.started
}

// monitorStart by polling server until it returns a 200 at /upcheck.
func (w *Web3RemoteSigner) monitorStart() {
	client := &http.Client{}
	for {
		req, err := http.NewRequestWithContext(w.ctx, "GET", fmt.Sprintf("http://localhost:%d/upcheck", Web3RemoteSignerPort), nil)
		if err != nil {
			panic(err)
		}
		res, err := client.Do(req)
		_ = err
		if res != nil && res.StatusCode == 200 {
			close(w.started)
			return
		}
		time.Sleep(time.Second)
	}
}

func (w *Web3RemoteSigner) wait(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	case <-w.ctx.Done():
		return
	case <-w.started:
		return
	}
}

// PublicKeys queries the web3signer and returns the response keys.
func (w *Web3RemoteSigner) PublicKeys(ctx context.Context) ([]bls.PublicKey, error) {
	w.wait(ctx)

	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://localhost:%d/api/v1/eth2/publicKeys", Web3RemoteSignerPort), nil)
	if err != nil {
		return nil, err
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("returned status code %d", res.StatusCode)
	}
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	} else if len(b) == 0 {
		return nil, errors.New("no response body")
	}
	var keys []string
	if err := json.Unmarshal(b, &keys); err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return nil, errors.New("no keys returned")
	}

	var pks []bls.PublicKey
	for _, key := range keys {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		raw, err := hex.DecodeString(strings.TrimPrefix(key, "0x"))
		if err != nil {
			return nil, err
		}
		pk, err := bls.PublicKeyFromBytes(raw)
		if err != nil {
			return nil, err
		}
		pks = append(pks, pk)
	}
	return pks, nil
}

const keystoreYamlFormat = `type: "%s"
keyType: "%s"
privateKey: "%s"
`

func writeKeystoreKeys(ctx context.Context, keystorePath string, numKeys uint64) error {
	if err := os.MkdirAll(keystorePath, 0750); err != nil {
		return err
	}

	priv, pub, err := interop.DeterministicallyGenerateKeys(0, numKeys)
	if err != nil {
		return err
	}
	for i, pk := range priv {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		rkf := &rawKeyFile{
			Type:       "file-raw",
			KeyType:    "BLS",
			PrivateKey: "0x" + hex.EncodeToString(pk.Marshal()),
		}
		// Note: yaml.v2 does not support marshalling yaml while enforcing yaml strings to be
		// quoted with double quotes. Consensys' web3signer fails to read keystores without the
		// double quotes. As such, we are using a template to build the yaml instead of
		// yaml.Marshal().
		// See: https://github.com/go-yaml/yaml/issues/556
		// See: https://github.com/ConsenSys/web3signer/issues/485
		b := []byte(fmt.Sprintf(keystoreYamlFormat, rkf.Type, rkf.KeyType, rkf.PrivateKey))
		if err := os.WriteFile(path.Join(keystorePath, fmt.Sprintf("key-0x%s.yaml", hex.EncodeToString(pub[i].Marshal()))), b, 0600); err != nil {
			return err
		}
	}

	return nil
}
