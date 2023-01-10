package remote_web3signer

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/crypto/bls"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpbservice "github.com/prysmaticlabs/prysm/v3/proto/eth/service"
	validatorpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/validator/keymanager/remote-web3signer/internal"
	"github.com/prysmaticlabs/prysm/v3/validator/keymanager/remote-web3signer/v1/mock"
)

var validClientCert = `-----BEGIN CERTIFICATE-----
MIIEITCCAgmgAwIBAgIQXUJWQZgVO4IX+zlWGI1/mTANBgkqhkiG9w0BAQsFADAU
MRIwEAYDVQQDEwlBdHRlc3RhbnQwHhcNMjAwMzE3MDgwNjU3WhcNMjEwOTE3MDc1
OTUyWjASMRAwDgYDVQQDEwdjbGllbnQxMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A
MIIBCgKCAQEAsc977g16Tan2j7YuA+zQOlDntb4Bkfs4sDOznOEvnozHwRZOgfcP
jVcA9AS5eZOGIRrsTssptrgVNDPoIHWoKk7LAKyyLM3dGp5PWeyMBoQA5cq+yPAT
4JkJpDnBFfwxXB99osJH0z3jSTRa62CSVvPRBisK4B9AlLQfcleEQlKJugy9tOAj
G7zodwEi+J4AYQHmOiwL38ZsKq9We5y4HMQ0E7de0FoU5QHrtuPNrTuwVwrq825l
cEAAFey6Btngx+sziysPHWHYOq4xOZ1UPBApeaAFLguzusc/4VwM7kzRNr4VOD8a
eC3CtKLhBBVVxHI5ZlaHS+YylNGYD4+FxQIDAQABo3EwbzAOBgNVHQ8BAf8EBAMC
A7gwHQYDVR0lBBYwFAYIKwYBBQUHAwEGCCsGAQUFBwMCMB0GA1UdDgQWBBQDGCE0
3k4rHzB+Ycf3pt1MzeDPgzAfBgNVHSMEGDAWgBScIYZa4dQBIW/gVwR0ctGCuHhe
9jANBgkqhkiG9w0BAQsFAAOCAgEAHG/EfvqIwbhYfci+zRCYC7aQPuvhivJblBwN
mbXo2qsxvje1hcKm0ptJLOy/cjJzeLJYREhQlXDPRJC/xgELnbXRjgag82r35+pf
wVJwP6Yw53VCM3o0QKsUrKyMm4sAijOBrJyqpB5untAieZsry5Bfj0S4YobbtdJa
VsEioU07fVVczf5lYN0XrLgRnXq3LMkTiZ6drFiqLkwmXQZVxNujmcaFSm7yCALl
EdhYNmaqedS5me5UOGxwPacrsZwWF9dvMsl3OswgTcaGdsUtx2/q+S2vbZUAM/Gw
qaTanDfvVtVTF7KzVN9hiqKe4mO0HHHK2HWJYBLdRJjInOgRW+53hCmUhLxD+Dq+
31jLKxn/Y4hyH9E+55b1sJHCFpsbEtVD53fojiH2C/uLbhq4Wr1PXgOoxzf2KeSQ
B3ENu8C4b6AlNhqOnz5zeDcx8Ug0vMfVDAwf6RAYMG5b/MoWNKcLNXhk8H1nbVkt
16ppjh6I27JqfNqfP2J/p3BF++ZugZuWfN9DRaJ6UPz+yyF7eW8fyDAQNl7LS0Kh
8PlF5cYvyIIKVHe38Mn8ZAWboKUs0xNv2vhA9V/4Q1ZzAEkXjmbk8H26sjGvJnvg
Lgm/+6LVWR4EnUlU8aEWASEpTWq2lSRF3ZOvNstHnufyiDfcwDcl/IKKQiVQQ3mX
tw8Jf74=
-----END CERTIFICATE-----`

// skipcq: SCT-1000
var validClientKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAsc977g16Tan2j7YuA+zQOlDntb4Bkfs4sDOznOEvnozHwRZO
gfcPjVcA9AS5eZOGIRrsTssptrgVNDPoIHWoKk7LAKyyLM3dGp5PWeyMBoQA5cq+
yPAT4JkJpDnBFfwxXB99osJH0z3jSTRa62CSVvPRBisK4B9AlLQfcleEQlKJugy9
tOAjG7zodwEi+J4AYQHmOiwL38ZsKq9We5y4HMQ0E7de0FoU5QHrtuPNrTuwVwrq
825lcEAAFey6Btngx+sziysPHWHYOq4xOZ1UPBApeaAFLguzusc/4VwM7kzRNr4V
OD8aeC3CtKLhBBVVxHI5ZlaHS+YylNGYD4+FxQIDAQABAoIBAQCjV2MVcDQmHDhw
FH95A5bVu3TgM8flfs64rwYU25iPIexuqDs+kOMsh/xMLfrkgGz7BGyIhYGwZLK1
3ekjyHHPS8qYuAyFtCelSEDE7tRDOAhLEFDq7gCUloGQ561EsQP3CMa1OZwZpgSh
PwM2ruRAFIK0E95NvOfqsv0gYN0Svo7hYjNsvW6ok/ZGMyN2ikcRR04wGOFOGjfT
xTmfURc9ejnOjHAOqLTpToPwM1/gWWR2iMQefC4njy4MO2BXqOPUmHxmmR4PYhu2
8EcKbyRs+/fvL3GgD3VAlOe5vnkfBzssQhHmexgSk5lHZrcSxUGXYGrYKPAeV2mk
5HRBWp0RAoGBAOUn5w+NCAugcTGP0hfNlyGXsXqUZvnMyFWvUcxgzgPlJyEyDnKn
aIb1DFOF2HckCfLZdrHqqgaF6K3TDvW9BgSKIsvISpo1S95ZPD6DKUo6YQ10CQRW
q/ZZVbxtFksVgFRGYpCVmPNULmx7CiXDT1b/suwNMAwCZwiNPTSvKQVLAoGBAMaj
zDo1/eepRslqnz5s8hh7dGEjfG/ZJcLgAJAxCyAgnIP4Tls7QkNhCVp9LcN6i1bc
CnT6AIuZRXSJWEdp4k2QnVFUmh9Q5MGgwrKYSY5M/1puTISlF1yQ8J6FX8BlDVmy
4dyaSyC0RIvgBzF9/KBDxxmJcHgGQ0awLeeyl4cvAoGBAN83FS3itLmOmXQrofyp
uNNyDeFXeU9OmL5OPqGUkljc+Favib9JLtp3DIC3WfoD0uUJy0LXULN18QaRFnts
mtYFMIvMGE9KJxL5XWOPI8M4Rp1yL+5X9r3Km2cl45dT5GMzBIPOFOTBVU86MtJC
A6C9Bi5FUk4AcRi1a69MB+stAoGAWNiwoyS9IV38dGCFQ4W1LzAg2MXnhZuJoUVR
2yykfkU33Gs2mOXDeKGxblDpJDLumfYnkzSzA72VbE92NdLtTqYtR1Bg8zraZqTC
EOG+nLBh0o/dF8ND1LpbdXvQXRyVwRYaofI9Qi5/LlUQwplIYmKObiSkMnsSok5w
6d5emi8CgYBjtUihOFaAmgqkTHOn4j4eKS1O7/H8QQSVe5M0bocmAIbgJ4At3GnI
E1JcIY2SZtSwAWs6aQPGE42gwsNCCsQWdJNtViO23JbCwlcPToC4aDfc0JJNaYqp
oVV7C5jmJh9VRd2tXIXIZMMNOfThfNf2qDQuJ1S2t5KugozFiRsHUg==
-----END RSA PRIVATE KEY-----`

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
func (mc *MockClient) GetPublicKeys(_ context.Context, _ string) ([][48]byte, error) {
	var keys [][48]byte
	for _, pk := range mc.PublicKeys {
		decoded, err := hex.DecodeString(strings.TrimPrefix(pk, "0x"))
		if err != nil {
			return nil, err
		}
		keys = append(keys, bytesutil.ToBytes48(decoded))
	}
	if mc.isThrowingError {
		return nil, fmt.Errorf("mock error")
	}
	return keys, nil
}

func TestKeymanager_Sign(t *testing.T) {
	client := &MockClient{
		Signature: "0xb3baa751d0a9132cfe93e4e3d5ff9075111100e3789dca219ade5a24d27e19d16b3353149da1833e9b691bb38634e8dc04469be7032132906c927d7e1a49b414730612877bc6b2810c8f202daf793d1ab0d6b5cb21d52f9e52e883859887a5d9",
	}
	ctx := context.Background()
	root, err := hexutil.Decode("0x270d43e74ce340de4bca2b1936beca0f4f5408d9e78aec4850920baf659d5b69")
	if err != nil {
		fmt.Printf("error: %v", err)
	}
	config := &SetupConfig{
		BaseEndpoint:          "http://example.com",
		GenesisValidatorsRoot: root,
		PublicKeysURL:         "http://example2.com/api/v1/eth2/publicKeys",
	}
	km, err := NewKeymanager(ctx, config)
	if err != nil {
		fmt.Printf("error: %v", err)
	}
	km.client = client
	desiredSigBytes, err := hexutil.Decode(client.Signature)
	if err != nil {
		fmt.Printf("error: %v", err)
	}
	desiredSig, err := bls.SignatureFromBytes(desiredSigBytes)
	if err != nil {
		fmt.Printf("error: %v", err)
	}
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
			name: "AGGREGATE_AND_PROOF",
			args: args{
				request: mock.GetMockSignRequest("AGGREGATE_AND_PROOF"),
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
				t.Errorf("GetVoluntaryExitSignRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.DeepEqual(t, got, tt.want)
		})
	}

}

func TestKeymanager_FetchValidatingPublicKeys_HappyPath_WithKeyList(t *testing.T) {
	ctx := context.Background()
	decodedKey, err := hexutil.Decode("0xa2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820")
	if err != nil {
		fmt.Printf("error: %v", err)
	}
	keys := [][48]byte{
		bytesutil.ToBytes48(decodedKey),
	}
	root, err := hexutil.Decode("0x270d43e74ce340de4bca2b1936beca0f4f5408d9e78aec4850920baf659d5b69")
	if err != nil {
		fmt.Printf("error: %v", err)
	}
	config := &SetupConfig{
		BaseEndpoint:          "http://example.com",
		GenesisValidatorsRoot: root,
		ProvidedPublicKeys:    keys,
	}
	km, err := NewKeymanager(ctx, config)
	if err != nil {
		fmt.Printf("error: %v", err)
	}
	resp, err := km.FetchValidatingPublicKeys(ctx)
	if err != nil {
		fmt.Printf("error: %v", err)
	}
	assert.NotNil(t, resp)
	assert.NoError(t, err)
	assert.Equal(t, resp, keys)
}

func TestKeymanager_FetchValidatingPublicKeys_HappyPath_WithExternalURL(t *testing.T) {
	ctx := context.Background()
	client := &MockClient{
		PublicKeys: []string{"0xa2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820"},
	}
	decodedKey, err := hexutil.Decode("0xa2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820")
	if err != nil {
		fmt.Printf("error: %v", err)
	}
	keys := [][48]byte{
		bytesutil.ToBytes48(decodedKey),
	}
	root, err := hexutil.Decode("0x270d43e74ce340de4bca2b1936beca0f4f5408d9e78aec4850920baf659d5b69")
	if err != nil {
		fmt.Printf("error: %v", err)
	}
	config := &SetupConfig{
		BaseEndpoint:          "http://example.com",
		GenesisValidatorsRoot: root,
		PublicKeysURL:         "http://example2.com/api/v1/eth2/publicKeys",
	}
	km, err := NewKeymanager(ctx, config)
	if err != nil {
		fmt.Printf("error: %v", err)
	}
	km.client = client
	resp, err := km.FetchValidatingPublicKeys(ctx)
	if err != nil {
		fmt.Printf("error: %v", err)
	}
	assert.NotNil(t, resp)
	assert.NoError(t, err)
	assert.Equal(t, resp, keys)
}

func TestKeymanager_FetchValidatingPublicKeys_WithExternalURL_ThrowsError(t *testing.T) {
	ctx := context.Background()
	client := &MockClient{
		PublicKeys:      []string{"0xa2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820"},
		isThrowingError: true,
	}
	root, err := hexutil.Decode("0x270d43e74ce340de4bca2b1936beca0f4f5408d9e78aec4850920baf659d5b69")
	if err != nil {
		fmt.Printf("error: %v", err)
	}
	config := &SetupConfig{
		BaseEndpoint:          "http://example.com",
		GenesisValidatorsRoot: root,
		PublicKeysURL:         "http://example2.com/api/v1/eth2/publicKeys",
	}
	km, err := NewKeymanager(ctx, config)
	if err != nil {
		fmt.Printf("error: %v", err)
	}
	km.client = client
	resp, err := km.FetchValidatingPublicKeys(ctx)
	assert.NotNil(t, err)
	assert.Equal(t, nil, resp)
	assert.Equal(t, "could not get public keys from remote server url: http://example2.com/api/v1/eth2/publicKeys: mock error", fmt.Sprintf("%v", err))
}

func TestKeymanager_AddPublicKeys(t *testing.T) {
	ctx := context.Background()
	root, err := hexutil.Decode("0x270d43e74ce340de4bca2b1936beca0f4f5408d9e78aec4850920baf659d5b69")
	if err != nil {
		fmt.Printf("error: %v", err)
	}
	config := &SetupConfig{
		BaseEndpoint:          "http://example.com",
		GenesisValidatorsRoot: root,
	}
	km, err := NewKeymanager(ctx, config)
	if err != nil {
		fmt.Printf("error: %v", err)
	}
	pubkey, err := hexutil.Decode("0xa2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820")
	require.NoError(t, err)
	publicKeys := [][fieldparams.BLSPubkeyLength]byte{
		bytesutil.ToBytes48(pubkey),
	}
	statuses, err := km.AddPublicKeys(ctx, publicKeys)
	require.NoError(t, err)
	for _, status := range statuses {
		require.Equal(t, ethpbservice.ImportedRemoteKeysStatus_IMPORTED, status.Status)
	}
	statuses, err = km.AddPublicKeys(ctx, publicKeys)
	require.NoError(t, err)
	for _, status := range statuses {
		require.Equal(t, ethpbservice.ImportedRemoteKeysStatus_DUPLICATE, status.Status)
	}
}

func TestKeymanager_DeletePublicKeys(t *testing.T) {
	ctx := context.Background()
	root, err := hexutil.Decode("0x270d43e74ce340de4bca2b1936beca0f4f5408d9e78aec4850920baf659d5b69")
	if err != nil {
		fmt.Printf("error: %v", err)
	}
	config := &SetupConfig{
		BaseEndpoint:          "http://example.com",
		GenesisValidatorsRoot: root,
	}
	km, err := NewKeymanager(ctx, config)
	if err != nil {
		fmt.Printf("error: %v", err)
	}
	pubkey, err := hexutil.Decode("0xa2b5aaad9c6efefe7bb9b1243a043404f3362937cfb6b31833929833173f476630ea2cfeb0d9ddf15f97ca8685948820")
	require.NoError(t, err)
	publicKeys := [][fieldparams.BLSPubkeyLength]byte{
		bytesutil.ToBytes48(pubkey),
	}
	statuses, err := km.AddPublicKeys(ctx, publicKeys)
	require.NoError(t, err)
	for _, status := range statuses {
		require.Equal(t, ethpbservice.ImportedRemoteKeysStatus_IMPORTED, status.Status)
	}

	s, err := km.DeletePublicKeys(ctx, publicKeys)
	require.NoError(t, err)
	for _, status := range s {
		require.Equal(t, ethpbservice.DeletedRemoteKeysStatus_DELETED, status.Status)
	}

	s, err = km.DeletePublicKeys(ctx, publicKeys)
	require.NoError(t, err)
	for _, status := range s {
		require.Equal(t, ethpbservice.DeletedRemoteKeysStatus_NOT_FOUND, status.Status)
	}
}

func TestNewKeymanager_Certificates(t *testing.T) {
	tests := []struct {
		name       string
		opts       *SetupConfig
		clientCert string
		clientKey  string
		caCert     string
		err        string
	}{
		{
			name: "NoClientCertificate",
			opts: &SetupConfig{

				RequireTLS: true,
			},
			err: "client certificate is required",
		},
		{
			name: "NoClientKey",
			opts: &SetupConfig{

				RequireTLS:     true,
				ClientCertPath: "/foo/client.crt",
				ClientKeyPath:  "",
			},
			err: "client key is required",
		},
		{
			name: "MissingClientKey",
			opts: &SetupConfig{
				RequireTLS:     true,
				ClientCertPath: "/foo/client.crt",
				ClientKeyPath:  "/foo/client.key",
				CACertPath:     "",
			},
			err: "failed to obtain client's certificate and/or key",
		},
		{
			name:       "BadClientCert",
			clientCert: `bad`,
			clientKey:  validClientKey,
			opts: &SetupConfig{
				RequireTLS: true,
			},
			err: "failed to obtain client's certificate and/or key: tls: failed to find any PEM data in certificate input",
		},
		{
			name:       "BadClientKey",
			clientCert: validClientCert,
			clientKey:  `bad`,
			opts: &SetupConfig{
				RequireTLS: true,
			},
			err: "failed to obtain client's certificate and/or key: tls: failed to find any PEM data in key input",
		},
		{
			name:       "MissingCACert",
			clientCert: validClientCert,
			clientKey:  validClientKey,
			opts: &SetupConfig{

				RequireTLS: true,
				CACertPath: `bad`,
			},
			err: "failed to obtain server's CA certificate: open bad: no such file or directory",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, err := hexutil.Decode("0x270d43e74ce340de4bca2b1936beca0f4f5408d9e78aec4850920baf659d5b69")
			if err != nil {
				fmt.Printf("error: %v", err)
			}
			if test.caCert != "" || test.clientCert != "" || test.clientKey != "" {
				dir := fmt.Sprintf("%s/%s", t.TempDir(), test.name)
				require.NoError(t, os.MkdirAll(dir, 0777))
				if test.caCert != "" {
					caCertPath := fmt.Sprintf("%s/ca.crt", dir)
					err := os.WriteFile(caCertPath, []byte(test.caCert), params.BeaconIoConfig().ReadWritePermissions)
					require.NoError(t, err, "Failed to write CA certificate")
					test.opts.CACertPath = caCertPath
				}
				if test.clientCert != "" {
					clientCertPath := fmt.Sprintf("%s/client.crt", dir)
					err := os.WriteFile(clientCertPath, []byte(test.clientCert), params.BeaconIoConfig().ReadWritePermissions)
					require.NoError(t, err, "Failed to write client certificate")
					test.opts.ClientCertPath = clientCertPath
				}
				if test.clientKey != "" {
					clientKeyPath := fmt.Sprintf("%s/client.key", dir)
					err := os.WriteFile(clientKeyPath, []byte(test.clientKey), params.BeaconIoConfig().ReadWritePermissions)
					require.NoError(t, err, "Failed to write client key")
					test.opts.ClientKeyPath = clientKeyPath
				}
			}
			_, err = NewKeymanager(context.Background(), &SetupConfig{
				BaseEndpoint:          "http://example.com",
				GenesisValidatorsRoot: root,
				PublicKeysURL:         "http://example2.com/api/v1/eth2/publicKeys",
				RequireTLS:            test.opts.RequireTLS,
				ClientCertPath:        test.opts.ClientCertPath,
				ClientKeyPath:         test.opts.ClientKeyPath,
				CACertPath:            test.opts.CACertPath,
			})
			if test.err == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, test.err, err)
			}
		})
	}
}
