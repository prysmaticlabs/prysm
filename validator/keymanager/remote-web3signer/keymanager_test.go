package remote_web3signer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"testing"
	"time"

	fieldparams "github.com/OffchainLabs/prysm/v7/config/fieldparams"
	"github.com/OffchainLabs/prysm/v7/crypto/bls"
	"github.com/OffchainLabs/prysm/v7/encoding/bytesutil"
	"github.com/OffchainLabs/prysm/v7/io/file"
	validatorpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1/validator-client"
	"github.com/OffchainLabs/prysm/v7/testing/require"
	"github.com/OffchainLabs/prysm/v7/validator/keymanager"
	"github.com/OffchainLabs/prysm/v7/validator/keymanager/remote-web3signer/internal"
	"github.com/OffchainLabs/prysm/v7/validator/keymanager/remote-web3signer/types/mock"
	"github.com/ethereum/go-ethereum/common/hexutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

type MockClient struct {
	Signature       string
	PublicKeys      []string
	isThrowingError bool
}

func (mc *MockClient) Sign(_ context.Context, _ string, _ internal.SignRequestJson) (bls.Signature, error) {
	decoded, err := hexutil.Decode(mc.Signature)
	if err != nil {
		return nil, err
	}
	return bls.SignatureFromBytes(decoded)
}
func (mc *MockClient) GetPublicKeys(_ context.Context, _ string) ([]string, error) {
	return mc.PublicKeys, nil
}

func newFileKeymanager(t *testing.T, contents string) (*Keymanager, string, chan [][fieldparams.BLSPubkeyLength]byte) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	keyFilePath := filepath.Join(t.TempDir(), "keyfile.txt")
	require.NoError(t, file.WriteFile(keyFilePath, []byte(contents)))
	root, err := hexutil.Decode("0x270d43e74ce340de4bca2b1936beca0f4f5408d9e78aec4850920baf659d5b69")
	require.NoError(t, err)
	km, err := NewKeymanager(ctx, &SetupConfig{
		BaseEndpoint:          "http://example.com",
		GenesisValidatorsRoot: root,
		KeyFilePath:           keyFilePath,
	})
	require.NoError(t, err)

	changes := make(chan [][fieldparams.BLSPubkeyLength]byte, 10)
	sub := km.SubscribeAccountChanges(changes)
	t.Cleanup(sub.Unsubscribe)
	return km, keyFilePath, changes
}

func requireAccountChange(t *testing.T, changes <-chan [][fieldparams.BLSPubkeyLength]byte, want ...string) {
	t.Helper()
	select {
	case keys := <-changes:
		got := make([]string, len(keys))
		for i, key := range keys {
			got[i] = hexutil.Encode(key[:])
		}
		slices.Sort(got)
		slices.Sort(want)
		if !slices.Equal(want, got) {
			t.Fatalf("unexpected account change: got %v, want %v", got, want)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for account change")
	}
}

func requireNoAccountChange(t *testing.T, changes <-chan [][fieldparams.BLSPubkeyLength]byte) {
	t.Helper()
	select {
	case keys := <-changes:
		t.Fatalf("unexpected additional account change: %#x", keys)
	default:
	}
}

func TestNewKeymanager(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode([]string{"0xa2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820"})
		require.NoError(t, err)
	}))
	root, err := hexutil.Decode("0x270d43e74ce340de4bca2b1936beca0f4f5408d9e78aec4850920baf659d5b69")
	require.NoError(t, err)
	tests := []struct {
		name         string
		args         *SetupConfig
		fileContents []string
		want         []string
		wantErr      string
		wantLog      string
	}{
		{
			name: "happy path public key url",
			args: &SetupConfig{
				BaseEndpoint:          "http://prysm.xyz/",
				GenesisValidatorsRoot: root,
				PublicKeysURL:         srv.URL + "/public_keys",
			},
			want: []string{"0xa2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820"},
		},
		{
			name: "bad public key url",
			args: &SetupConfig{
				BaseEndpoint:          "http://prysm.xyz/",
				GenesisValidatorsRoot: root,
				PublicKeysURL:         "0x270d43e74ce340de4bca2b1936beca0f4f5408d9e78aec4850920baf659d5b69",
			},
			wantErr: "could not get public keys from remote server URL",
		},
		{
			name: "happy path provided public keys",
			args: &SetupConfig{
				BaseEndpoint:          "http://prysm.xyz/",
				GenesisValidatorsRoot: root,
				ProvidedPublicKeys:    []string{"0xa2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820"},
			},
			want: []string{"0xa2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820"},
		},
		{
			name: "path provided public keys, some bad key",
			args: &SetupConfig{
				BaseEndpoint:          "http://prysm.xyz/",
				GenesisValidatorsRoot: root,
				ProvidedPublicKeys:    []string{"0xa2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820", "http://prysm.xyz/"},
			},
			wantErr: "could not decode public key",
		},
		{
			name: "path provided public keys, some bad hex for key",
			args: &SetupConfig{
				BaseEndpoint:          "http://prysm.xyz/",
				GenesisValidatorsRoot: root,
				ProvidedPublicKeys:    []string{"0xa2b5aaad9c6efefe7bb9b1243a043404f3362937"},
			},
			wantErr: "has invalid length",
		},
		{
			name: "key file with malformed line",
			args: &SetupConfig{
				BaseEndpoint:          "http://prysm.xyz/",
				GenesisValidatorsRoot: root,
				KeyFilePath:           filepath.Join(t.TempDir(), "bad_keyfile.txt"),
			},
			fileContents: []string{"8000a9a6d3f5e22d783eefaadbcf0298146adb5d95b04db910a0d4e16976b30229d0b1e7b9cda6c7e0bfa11f72efe055", "not-a-key"},
			want:         []string{"0x8000a9a6d3f5e22d783eefaadbcf0298146adb5d95b04db910a0d4e16976b30229d0b1e7b9cda6c7e0bfa11f72efe055"},
			wantLog:      "Invalid public key in remote signer key file",
		},
		{
			name: "happy path key file",
			args: &SetupConfig{
				BaseEndpoint:          "http://prysm.xyz/",
				GenesisValidatorsRoot: root,
				KeyFilePath:           filepath.Join(t.TempDir(), "good_keyfile.txt"),
			},
			fileContents: []string{"8000a9a6d3f5e22d783eefaadbcf0298146adb5d95b04db910a0d4e16976b30229d0b1e7b9cda6c7e0bfa11f72efe055", "0x800057e262bfe42413c2cfce948ff77f11efeea19721f590c8b5b2f32fecb0e164cafba987c80465878408d05b97c9be"},
			want:         []string{"0x8000a9a6d3f5e22d783eefaadbcf0298146adb5d95b04db910a0d4e16976b30229d0b1e7b9cda6c7e0bfa11f72efe055", "0x800057e262bfe42413c2cfce948ff77f11efeea19721f590c8b5b2f32fecb0e164cafba987c80465878408d05b97c9be"},
		},
		{
			name: "happy path public key url with good keyfile",
			args: &SetupConfig{
				BaseEndpoint:          "http://prysm.xyz/",
				GenesisValidatorsRoot: root,
				PublicKeysURL:         srv.URL + "/public_keys",
				KeyFilePath:           filepath.Join(t.TempDir(), "good_keyfile.txt"),
			},
			fileContents: []string{"0x8000a9a6d3f5e22d783eefaadbcf0298146adb5d95b04db910a0d4e16976b30229d0b1e7b9cda6c7e0bfa11f72efe055", "800057e262bfe42413c2cfce948ff77f11efeea19721f590c8b5b2f32fecb0e164cafba987c80465878408d05b97c9be"},
			want:         []string{"0xa2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820", "0x8000a9a6d3f5e22d783eefaadbcf0298146adb5d95b04db910a0d4e16976b30229d0b1e7b9cda6c7e0bfa11f72efe055", "0x800057e262bfe42413c2cfce948ff77f11efeea19721f590c8b5b2f32fecb0e164cafba987c80465878408d05b97c9be"},
		},
		{
			name: "happy path provided public keys with good keyfile",
			args: &SetupConfig{
				BaseEndpoint:          "http://prysm.xyz/",
				GenesisValidatorsRoot: root,
				KeyFilePath:           filepath.Join(t.TempDir(), "good_keyfile.txt"),
				ProvidedPublicKeys:    []string{"0xa2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820"},
			},
			fileContents: []string{"0x8000a9a6d3f5e22d783eefaadbcf0298146adb5d95b04db910a0d4e16976b30229d0b1e7b9cda6c7e0bfa11f72efe055", "800057e262bfe42413c2cfce948ff77f11efeea19721f590c8b5b2f32fecb0e164cafba987c80465878408d05b97c9be"},
			want:         []string{"0xa2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820", "0x8000a9a6d3f5e22d783eefaadbcf0298146adb5d95b04db910a0d4e16976b30229d0b1e7b9cda6c7e0bfa11f72efe055", "0x800057e262bfe42413c2cfce948ff77f11efeea19721f590c8b5b2f32fecb0e164cafba987c80465878408d05b97c9be"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logHook := logTest.NewGlobal()
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if tt.args.KeyFilePath != "" && len(tt.fileContents) != 0 {
				bytesBuf := new(bytes.Buffer)
				for _, content := range tt.fileContents {
					_, err := bytesBuf.WriteString(content) // test without 0x
					require.NoError(t, err)
					_, err = bytesBuf.WriteString("\n")
					require.NoError(t, err)
				}
				err = file.WriteFile(tt.args.KeyFilePath, bytesBuf.Bytes())
				require.NoError(t, err)
			}

			km, err := NewKeymanager(ctx, tt.args)
			if tt.wantLog != "" {
				require.LogsContain(t, logHook, tt.wantLog)
			}
			if tt.wantErr != "" {
				require.ErrorContains(t, tt.wantErr, err)
				return
			}
			require.NoError(t, err)
			keys := make([]string, len(km.providedPublicKeys))
			for i, key := range km.providedPublicKeys {
				keys[i] = hexutil.Encode(key[:])
			}
			slices.Sort(keys)
			slices.Sort(tt.want)
			require.DeepEqual(t, tt.want, keys)
		})
	}
}

func TestNewKeyManager_fileMissing(t *testing.T) {
	keyFilePath := filepath.Join(t.TempDir(), "keyfile.txt")
	root, err := hexutil.Decode("0x270d43e74ce340de4bca2b1936beca0f4f5408d9e78aec4850920baf659d5b69")
	require.NoError(t, err)
	_, err = NewKeymanager(context.TODO(), &SetupConfig{
		BaseEndpoint:          "http://example.com",
		GenesisValidatorsRoot: root,
		KeyFilePath:           keyFilePath,
		ProvidedPublicKeys:    []string{"0x800077e04f8d7496099b3d30ac5430aea64873a45e5bcfe004d2095babcbf55e21138ff0d5691abc29da190aa32755c6"},
	})
	require.ErrorContains(t, "no file exists in remote signer key file path", err)
}

func TestKeymanager_KeyFileChangesApplyOnRestart(t *testing.T) {
	const (
		keyA = "0xa2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820"
		keyB = "0x8000a9a6d3f5e22d783eefaadbcf0298146adb5d95b04db910a0d4e16976b30229d0b1e7b9cda6c7e0bfa11f72efe055"
	)

	tests := []struct {
		name   string
		change func(string) error
	}{
		{"replace", func(path string) error {
			if err := os.WriteFile(path+".tmp", []byte(keyB+"\n"), 0600); err != nil {
				return err
			}
			return os.Rename(path+".tmp", path)
		}},
		{"truncate", func(path string) error { return os.Truncate(path, 0) }},
		{"remove", os.Remove},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			km, keyFilePath, changes := newFileKeymanager(t, keyA+"\n")
			require.NoError(t, tt.change(keyFilePath))
			requireNoAccountChange(t, changes)
			keys, err := km.FetchValidatingPublicKeys(t.Context())
			require.NoError(t, err)
			require.Equal(t, 1, len(keys))
			require.Equal(t, keyA, hexutil.Encode(keys[0][:]))

			require.NoError(t, os.WriteFile(keyFilePath, []byte(keyB+"\n"), 0600))
			restarted, err := NewKeymanager(t.Context(), &SetupConfig{
				BaseEndpoint:          "http://example.com",
				GenesisValidatorsRoot: km.genesisValidatorsRoot,
				KeyFilePath:           keyFilePath,
			})
			require.NoError(t, err)
			keys, err = restarted.FetchValidatingPublicKeys(t.Context())
			require.NoError(t, err)
			require.Equal(t, 1, len(keys))
			require.Equal(t, keyB, hexutil.Encode(keys[0][:]))
		})
	}
}

func TestKeymanager_APIAccountNotifications(t *testing.T) {
	const (
		keyA = "0xa2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820"
		keyB = "0x8000a9a6d3f5e22d783eefaadbcf0298146adb5d95b04db910a0d4e16976b30229d0b1e7b9cda6c7e0bfa11f72efe055"
	)
	tests := []struct {
		changed  bool
		delete   bool
		status   keymanager.KeyStatusType
		name     string
		contents string
		key      string
		want     []string
	}{
		{true, false, keymanager.StatusImported, "import", "", keyA, []string{keyA}},
		{false, false, keymanager.StatusDuplicate, "duplicate", keyA + "\n", keyA, nil},
		{true, false, keymanager.StatusImported, "second import", keyA + "\n", keyB, []string{keyA, keyB}},
		{true, true, keymanager.StatusDeleted, "delete", keyA + "\n" + keyB + "\n", keyA, []string{keyB}},
		{false, true, keymanager.StatusNotFound, "missing", keyB + "\n", keyA, nil},
		{true, true, keymanager.StatusDeleted, "delete last", keyB + "\n", keyB, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			km, _, changes := newFileKeymanager(t, tt.contents)
			apply := km.AddPublicKeys
			if tt.delete {
				apply = km.DeletePublicKeys
			}
			statuses, err := apply([]string{tt.key})
			require.NoError(t, err)
			require.Equal(t, 1, len(statuses))
			require.Equal(t, tt.status, statuses[0].Status)
			if tt.changed {
				requireAccountChange(t, changes, tt.want...)
			}
			requireNoAccountChange(t, changes)
		})
	}
}

func TestKeymanager_ConcurrentAPIWritesPersist(t *testing.T) {
	const (
		keyA = "0xa2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820"
		keyB = "0x8000a9a6d3f5e22d783eefaadbcf0298146adb5d95b04db910a0d4e16976b30229d0b1e7b9cda6c7e0bfa11f72efe055"
	)
	km, keyFilePath, changes := newFileKeymanager(t, "")
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, key := range []string{keyA, keyB} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses, err := km.AddPublicKeys([]string{key})
			if err == nil && (len(statuses) != 1 || statuses[0].Status != keymanager.StatusImported) {
				err = fmt.Errorf("unexpected import statuses: %v", statuses)
			}
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	for err := range results {
		require.NoError(t, err)
	}
	select {
	case keys := <-changes:
		require.Equal(t, 1, len(keys))
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for first import")
	}
	requireAccountChange(t, changes, keyA, keyB)
	requireNoAccountChange(t, changes)
	restarted, err := NewKeymanager(t.Context(), &SetupConfig{
		BaseEndpoint:          "http://example.com",
		GenesisValidatorsRoot: km.genesisValidatorsRoot,
		KeyFilePath:           keyFilePath,
	})
	require.NoError(t, err)
	keys, err := restarted.FetchValidatingPublicKeys(t.Context())
	require.NoError(t, err)
	require.Equal(t, 2, len(keys))
	got := []string{hexutil.Encode(keys[0][:]), hexutil.Encode(keys[1][:])}
	slices.Sort(got)
	want := []string{keyA, keyB}
	slices.Sort(want)
	require.DeepEqual(t, want, got)
}

func TestKeymanager_DeleteAllFileKeysPersists(t *testing.T) {
	const key = "0xa2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820"
	km, keyFilePath, changes := newFileKeymanager(t, key+"\n")
	statuses, err := km.DeletePublicKeys([]string{key})
	require.NoError(t, err)
	require.Equal(t, keymanager.StatusDeleted, statuses[0].Status)
	requireAccountChange(t, changes)
	requireNoAccountChange(t, changes)
	contents, err := os.ReadFile(keyFilePath)
	require.NoError(t, err)
	require.Equal(t, 0, len(contents))
	restarted, err := NewKeymanager(t.Context(), &SetupConfig{
		BaseEndpoint:          "http://example.com",
		GenesisValidatorsRoot: km.genesisValidatorsRoot,
		KeyFilePath:           keyFilePath,
	})
	require.NoError(t, err)
	keys, err := restarted.FetchValidatingPublicKeys(t.Context())
	require.NoError(t, err)
	require.Equal(t, 0, len(keys))
}

func TestKeymanager_PersistenceErrors(t *testing.T) {
	keyA := hexutil.Encode(bytes.Repeat([]byte{0x11}, fieldparams.BLSPubkeyLength))
	keyB := hexutil.Encode(bytes.Repeat([]byte{0x22}, fieldparams.BLSPubkeyLength))
	tests := []struct {
		name      string
		want      []string
		delete    bool
		committed bool
	}{
		{"add before rename", []string{keyA}, false, false},
		{"delete before rename", []string{keyA}, true, false},
		{"add after rename", []string{keyA, keyB}, false, true},
		{"delete after rename", []string{}, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.committed && (runtime.GOOS == "windows" || os.Geteuid() == 0) {
				t.Skip("requires Unix directory permission enforcement")
			}
			km, keyFilePath, changes := newFileKeymanager(t, keyA+"\n")
			dir := filepath.Dir(keyFilePath)
			if tt.committed {
				t.Cleanup(func() { require.NoError(t, os.Chmod(dir, 0700)) })
				require.NoError(t, os.Chmod(dir, 0300))
			} else {
				require.NoError(t, os.RemoveAll(dir))
			}
			apply, key := km.AddPublicKeys, keyB
			if tt.delete {
				apply, key = km.DeletePublicKeys, keyA
			}
			_, err := apply([]string{key})
			require.ErrorContains(t, "persist remote signer key file", err)
			keys, err := km.FetchValidatingPublicKeys(t.Context())
			require.NoError(t, err)
			got := make([]string, len(keys))
			for i, key := range keys {
				got[i] = hexutil.Encode(key[:])
			}
			require.DeepEqual(t, tt.want, got)
			if tt.committed {
				requireAccountChange(t, changes, tt.want...)
				persisted, _, err := km.readKeyFile()
				require.NoError(t, err)
				require.DeepEqual(t, keys, persisted)
			}
			requireNoAccountChange(t, changes)
		})
	}
}

func TestKeymanager_APIPersistenceFollowsSymlink(t *testing.T) {
	const (
		keyA = "0xa2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820"
		keyB = "0x8000a9a6d3f5e22d783eefaadbcf0298146adb5d95b04db910a0d4e16976b30229d0b1e7b9cda6c7e0bfa11f72efe055"
	)
	target := filepath.Join(t.TempDir(), "target.txt")
	link := filepath.Join(t.TempDir(), "keys.txt")
	require.NoError(t, file.WriteFile(target, []byte(keyA+"\n")))
	require.NoError(t, os.Symlink(target, link))
	root, err := hexutil.Decode("0x270d43e74ce340de4bca2b1936beca0f4f5408d9e78aec4850920baf659d5b69")
	require.NoError(t, err)
	km, err := NewKeymanager(t.Context(), &SetupConfig{
		BaseEndpoint:          "http://example.com",
		GenesisValidatorsRoot: root,
		KeyFilePath:           link,
	})
	require.NoError(t, err)
	changes := make(chan [][fieldparams.BLSPubkeyLength]byte, 1)
	sub := km.SubscribeAccountChanges(changes)
	defer sub.Unsubscribe()
	statuses, err := km.AddPublicKeys([]string{keyB})
	require.NoError(t, err)
	require.Equal(t, keymanager.StatusImported, statuses[0].Status)
	requireAccountChange(t, changes, keyA, keyB)
	requireNoAccountChange(t, changes)
	info, err := os.Lstat(link)
	require.NoError(t, err)
	require.Equal(t, true, info.Mode()&os.ModeSymlink != 0)
	contents, err := os.ReadFile(target)
	require.NoError(t, err)
	require.Equal(t, keyB+"\n"+keyA+"\n", string(contents))
}

func TestNewKeyManager_FileAndFlagsWithDifferentKeys(t *testing.T) {
	const (
		fileKey = "0x8000a9a6d3f5e22d783eefaadbcf0298146adb5d95b04db910a0d4e16976b30229d0b1e7b9cda6c7e0bfa11f72efe055"
		flagKey = "0x800077e04f8d7496099b3d30ac5430aea64873a45e5bcfe004d2095babcbf55e21138ff0d5691abc29da190aa32755c6"
	)
	keyFilePath := filepath.Join(t.TempDir(), "keyfile.txt")
	require.NoError(t, file.WriteFile(keyFilePath, []byte(fileKey+"\n")))
	root, err := hexutil.Decode("0x270d43e74ce340de4bca2b1936beca0f4f5408d9e78aec4850920baf659d5b69")
	require.NoError(t, err)
	km, err := NewKeymanager(t.Context(), &SetupConfig{
		BaseEndpoint:          "http://example.com",
		GenesisValidatorsRoot: root,
		KeyFilePath:           keyFilePath,
		ProvidedPublicKeys:    []string{flagKey},
	})
	require.NoError(t, err)
	keys, err := km.FetchValidatingPublicKeys(t.Context())
	require.NoError(t, err)
	fileKeys, _, err := km.readKeyFile()
	require.NoError(t, err)
	require.DeepEqual(t, keys, fileKeys)
	require.Equal(t, 2, len(keys))
	require.Equal(t, flagKey, hexutil.Encode(keys[0][:]))
	require.Equal(t, fileKey, hexutil.Encode(keys[1][:]))
}

func TestReadKeyFile_PathMissing(t *testing.T) {
	root, err := hexutil.Decode("0x270d43e74ce340de4bca2b1936beca0f4f5408d9e78aec4850920baf659d5b69")
	require.NoError(t, err)

	require.NoError(t, err)
	km, err := NewKeymanager(context.TODO(), &SetupConfig{
		BaseEndpoint:          "http://example.com",
		GenesisValidatorsRoot: root,
	})
	require.NoError(t, err)
	_, _, err = km.readKeyFile()
	require.ErrorContains(t, "no key file path provided", err)
}

func TestKeymanager_Sign(t *testing.T) {
	client := &MockClient{
		Signature: "0xb3baa751d0a9132cfe93e4e3d5ff9075111100e3789dca219ade5a24d27e19d16b3353149da1833e9b691bb38634e8dc04469be7032132906c927d7e1a49b414730612877bc6b2810c8f202daf793d1ab0d6b5cb21d52f9e52e883859887a5d9",
	}
	ctx := t.Context()
	root, err := hexutil.Decode("0x270d43e74ce340de4bca2b1936beca0f4f5408d9e78aec4850920baf659d5b69")
	require.NoError(t, err)
	config := &SetupConfig{
		BaseEndpoint:          "http://example.com",
		GenesisValidatorsRoot: root,
	}
	km, err := NewKeymanager(ctx, config)
	require.NoError(t, err)
	km.client = client
	desiredSigBytes, err := hexutil.Decode(client.Signature)
	require.NoError(t, err)
	desiredSig, err := bls.SignatureFromBytes(desiredSigBytes)
	require.NoError(t, err)
	type args struct {
		request *validatorpb.SignRequest
	}
	tests := []struct {
		name    string
		args    args
		want    bls.Signature
		wantErr bool
	}{
		{
			name: "AGGREGATION_SLOT",
			args: args{
				request: mock.GetMockSignRequest("AGGREGATION_SLOT"),
			},
			want:    desiredSig,
			wantErr: false,
		},
		{
			name: "AGGREGATE_AND_PROOF_V2",
			args: args{
				request: mock.GetMockSignRequest("AGGREGATE_AND_PROOF_V2"),
			},
			want:    desiredSig,
			wantErr: false,
		},
		{
			name: "ATTESTATION",
			args: args{
				request: mock.GetMockSignRequest("ATTESTATION"),
			},
			want:    desiredSig,
			wantErr: false,
		},
		{
			name: "BLOCK",
			args: args{
				request: mock.GetMockSignRequest("BLOCK"),
			},
			want:    desiredSig,
			wantErr: false,
		},
		{
			name: "BLOCK_V2",
			args: args{
				request: mock.GetMockSignRequest("BLOCK_V2"),
			},
			want:    desiredSig,
			wantErr: false,
		},
		{
			name: "RANDAO_REVEAL",
			args: args{
				request: mock.GetMockSignRequest("RANDAO_REVEAL"),
			},
			want:    desiredSig,
			wantErr: false,
		},
		{
			name: "SYNC_COMMITTEE_CONTRIBUTION_AND_PROOF",
			args: args{
				request: mock.GetMockSignRequest("SYNC_COMMITTEE_CONTRIBUTION_AND_PROOF"),
			},
			want:    desiredSig,
			wantErr: false,
		},
		{
			name: "SYNC_COMMITTEE_MESSAGE",
			args: args{
				request: mock.GetMockSignRequest("SYNC_COMMITTEE_MESSAGE"),
			},
			want:    desiredSig,
			wantErr: false,
		},
		{
			name: "SYNC_COMMITTEE_SELECTION_PROOF",
			args: args{
				request: mock.GetMockSignRequest("SYNC_COMMITTEE_SELECTION_PROOF"),
			},
			want:    desiredSig,
			wantErr: false,
		},
		{
			name: "VOLUNTARY_EXIT",
			args: args{
				request: mock.GetMockSignRequest("VOLUNTARY_EXIT"),
			},
			want:    desiredSig,
			wantErr: false,
		},
		{
			name: "VALIDATOR_REGISTRATION",
			args: args{
				request: mock.GetMockSignRequest("VALIDATOR_REGISTRATION"),
			},
			want:    desiredSig,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := km.Sign(ctx, tt.args.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("name:%s error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			require.DeepEqual(t, got, tt.want)
		})
	}

}

func TestKeymanager_FetchValidatingPublicKeys_HappyPath_WithKeyList(t *testing.T) {
	ctx := t.Context()
	decodedKey, err := hexutil.Decode("0xa2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820")
	require.NoError(t, err)
	keys := [][48]byte{
		bytesutil.ToBytes48(decodedKey),
	}
	root, err := hexutil.Decode("0x270d43e74ce340de4bca2b1936beca0f4f5408d9e78aec4850920baf659d5b69")
	require.NoError(t, err)
	config := &SetupConfig{
		BaseEndpoint:          "http://example.com",
		GenesisValidatorsRoot: root,
		ProvidedPublicKeys:    []string{"0xa2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820"},
	}
	km, err := NewKeymanager(ctx, config)
	require.NoError(t, err)
	resp, err := km.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Nil(t, err)
	assert.EqualValues(t, resp, keys)
}

func TestKeymanager_FetchValidatingPublicKeys_HappyPath_WithExternalURL(t *testing.T) {
	ctx := t.Context()
	decodedKey, err := hexutil.Decode("0xa2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820")
	require.NoError(t, err)
	keys := [][48]byte{
		bytesutil.ToBytes48(decodedKey),
	}
	root, err := hexutil.Decode("0x270d43e74ce340de4bca2b1936beca0f4f5408d9e78aec4850920baf659d5b69")
	require.NoError(t, err)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode([]string{"0xa2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820"})
		require.NoError(t, err)
	}))
	defer srv.Close()
	config := &SetupConfig{
		BaseEndpoint:          "http://example.com",
		GenesisValidatorsRoot: root,
		PublicKeysURL:         srv.URL + "/api/v1/eth2/publicKeys",
	}
	km, err := NewKeymanager(ctx, config)
	require.NoError(t, err)
	resp, err := km.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.EqualValues(t, resp, keys)
}

func TestKeymanager_FetchValidatingPublicKeys_WithExternalURL_ThrowsError(t *testing.T) {
	ctx := t.Context()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, "mock error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	root, err := hexutil.Decode("0x270d43e74ce340de4bca2b1936beca0f4f5408d9e78aec4850920baf659d5b69")
	require.NoError(t, err)
	config := &SetupConfig{
		BaseEndpoint:          "http://example.com",
		GenesisValidatorsRoot: root,
		PublicKeysURL:         srv.URL + "/api/v1/eth2/publicKeys",
	}
	km, err := NewKeymanager(ctx, config)
	require.ErrorContains(t, fmt.Sprintf("could not get public keys from remote server URL %s/api/v1/eth2/publicKeys", srv.URL), err)
	assert.Nil(t, km)
}

func TestKeymanager_AddPublicKeys(t *testing.T) {
	ctx := t.Context()
	root, err := hexutil.Decode("0x270d43e74ce340de4bca2b1936beca0f4f5408d9e78aec4850920baf659d5b69")
	require.NoError(t, err)
	config := &SetupConfig{
		BaseEndpoint:          "http://example.com",
		GenesisValidatorsRoot: root,
	}
	km, err := NewKeymanager(ctx, config)
	require.NoError(t, err)
	publicKeys := []string{"0xa2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820"}
	statuses, err := km.AddPublicKeys(publicKeys)
	require.NoError(t, err)
	for _, status := range statuses {
		require.Equal(t, keymanager.StatusImported, status.Status)
	}
	statuses, err = km.AddPublicKeys(publicKeys)
	require.NoError(t, err)
	for _, status := range statuses {
		require.Equal(t, keymanager.StatusDuplicate, status.Status)
	}
}

func TestKeymanager_AddPublicKeys_WithFile(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	dir := t.TempDir()
	stdOutFile, err := os.Create(filepath.Clean(path.Join(dir, "keyfile.txt")))
	require.NoError(t, err)
	require.NoError(t, stdOutFile.Chmod(os.FileMode(0600)))
	keyFilePath := filepath.Join(dir, "keyfile.txt")
	root, err := hexutil.Decode("0x270d43e74ce340de4bca2b1936beca0f4f5408d9e78aec4850920baf659d5b69")
	require.NoError(t, err)
	config := &SetupConfig{
		BaseEndpoint:          "http://example.com",
		GenesisValidatorsRoot: root,
		KeyFilePath:           keyFilePath,
	}
	km, err := NewKeymanager(ctx, config)
	require.NoError(t, err)
	publicKeys := []string{"0xa2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820"}
	statuses, err := km.AddPublicKeys(publicKeys)
	require.NoError(t, err)
	for _, status := range statuses {
		require.Equal(t, keymanager.StatusImported, status.Status)
	}
	statuses, err = km.AddPublicKeys(publicKeys)
	require.NoError(t, err)
	for _, status := range statuses {
		require.Equal(t, keymanager.StatusDuplicate, status.Status)
	}
	keys, _, err := km.readKeyFile()
	require.NoError(t, err)
	require.Equal(t, len(keys), len(publicKeys))
	require.Equal(t, hexutil.Encode(keys[0][:]), publicKeys[0])
}

func TestKeymanager_DeletePublicKeys(t *testing.T) {
	ctx := t.Context()
	root, err := hexutil.Decode("0x270d43e74ce340de4bca2b1936beca0f4f5408d9e78aec4850920baf659d5b69")
	require.NoError(t, err)
	config := &SetupConfig{
		BaseEndpoint:          "http://example.com",
		GenesisValidatorsRoot: root,
	}
	km, err := NewKeymanager(ctx, config)
	require.NoError(t, err)
	publicKeys := []string{"0xa2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820"}
	statuses, err := km.AddPublicKeys(publicKeys)
	require.NoError(t, err)
	for _, status := range statuses {
		require.Equal(t, keymanager.StatusImported, status.Status)
	}

	s, err := km.DeletePublicKeys(publicKeys)
	require.NoError(t, err)
	for _, status := range s {
		require.Equal(t, keymanager.StatusDeleted, status.Status)
	}

	s, err = km.DeletePublicKeys(publicKeys)
	require.NoError(t, err)
	for _, status := range s {
		require.Equal(t, keymanager.StatusNotFound, status.Status)
	}
}

func TestKeymanager_DeletePublicKeys_WithFile(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	dir := t.TempDir()
	stdOutFile, err := os.Create(filepath.Clean(path.Join(dir, "keyfile.txt")))
	require.NoError(t, err)
	require.NoError(t, stdOutFile.Chmod(os.FileMode(0600)))
	keyFilePath := filepath.Join(dir, "keyfile.txt")
	root, err := hexutil.Decode("0x270d43e74ce340de4bca2b1936beca0f4f5408d9e78aec4850920baf659d5b69")
	require.NoError(t, err)
	config := &SetupConfig{
		BaseEndpoint:          "http://example.com",
		GenesisValidatorsRoot: root,
		KeyFilePath:           keyFilePath,
	}
	km, err := NewKeymanager(ctx, config)
	require.NoError(t, err)
	publicKeys := []string{"0xa2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820", "0x8000a9a6d3f5e22d783eefaadbcf0298146adb5d95b04db910a0d4e16976b30229d0b1e7b9cda6c7e0bfa11f72efe055"}
	statuses, err := km.AddPublicKeys(publicKeys)
	require.NoError(t, err)
	for _, status := range statuses {
		require.Equal(t, keymanager.StatusImported, status.Status)
	}

	s, err := km.DeletePublicKeys([]string{publicKeys[0]})
	require.NoError(t, err)
	for _, status := range s {
		require.Equal(t, keymanager.StatusDeleted, status.Status)
	}

	s, err = km.DeletePublicKeys([]string{publicKeys[0]})
	require.NoError(t, err)
	for _, status := range s {
		require.Equal(t, keymanager.StatusNotFound, status.Status)
	}
	keys, _, err := km.readKeyFile()
	require.NoError(t, err)
	require.Equal(t, len(keys), 1)
	require.Equal(t, hexutil.Encode(keys[0][:]), publicKeys[1])
}
