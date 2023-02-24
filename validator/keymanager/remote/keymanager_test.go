package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/prysmaticlabs/prysm/v3/async/event"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/crypto/bls"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	validatorpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/mock"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/validator/keymanager"
	logTest "github.com/sirupsen/logrus/hooks/test"
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

func TestNewRemoteKeymanager(t *testing.T) {
	tests := []struct {
		name       string
		opts       *KeymanagerOpts
		clientCert string
		clientKey  string
		caCert     string
		err        string
	}{
		{
			name: "NoCertificates",
			opts: &KeymanagerOpts{
				RemoteCertificate: nil,
			},
			err: "certificate configuration is missing",
		},
		{
			name: "NoClientCertificate",
			opts: &KeymanagerOpts{
				RemoteCertificate: &CertificateConfig{
					RequireTls: true,
				},
			},
			err: "client certificate is required",
		},
		{
			name: "NoClientKey",
			opts: &KeymanagerOpts{
				RemoteCertificate: &CertificateConfig{
					RequireTls:     true,
					ClientCertPath: "/foo/client.crt",
					ClientKeyPath:  "",
				},
			},
			err: "client key is required",
		},
		{
			name: "MissingClientKey",
			opts: &KeymanagerOpts{
				RemoteCertificate: &CertificateConfig{
					RequireTls:     true,
					ClientCertPath: "/foo/client.crt",
					ClientKeyPath:  "/foo/client.key",
					CACertPath:     "",
				},
			},
			err: "failed to obtain client's certificate and/or key",
		},
		{
			name:       "BadClientCert",
			clientCert: `bad`,
			clientKey:  validClientKey,
			opts: &KeymanagerOpts{
				RemoteCertificate: &CertificateConfig{
					RequireTls: true,
				},
			},
			err: "failed to obtain client's certificate and/or key: tls: failed to find any PEM data in certificate input",
		},
		{
			name:       "BadClientKey",
			clientCert: validClientCert,
			clientKey:  `bad`,
			opts: &KeymanagerOpts{
				RemoteCertificate: &CertificateConfig{
					RequireTls: true,
				},
			},
			err: "failed to obtain client's certificate and/or key: tls: failed to find any PEM data in key input",
		},
		{
			name:       "MissingCACert",
			clientCert: validClientCert,
			clientKey:  validClientKey,
			opts: &KeymanagerOpts{
				RemoteCertificate: &CertificateConfig{
					RequireTls: true,
					CACertPath: `bad`,
				},
			},
			err: "failed to obtain server's CA certificate: open bad: no such file or directory",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.caCert != "" || test.clientCert != "" || test.clientKey != "" {
				dir := fmt.Sprintf("%s/%s", t.TempDir(), test.name)
				require.NoError(t, os.MkdirAll(dir, 0777))
				if test.caCert != "" {
					caCertPath := fmt.Sprintf("%s/ca.crt", dir)
					err := os.WriteFile(caCertPath, []byte(test.caCert), params.BeaconIoConfig().ReadWritePermissions)
					require.NoError(t, err, "Failed to write CA certificate")
					test.opts.RemoteCertificate.CACertPath = caCertPath
				}
				if test.clientCert != "" {
					clientCertPath := fmt.Sprintf("%s/client.crt", dir)
					err := os.WriteFile(clientCertPath, []byte(test.clientCert), params.BeaconIoConfig().ReadWritePermissions)
					require.NoError(t, err, "Failed to write client certificate")
					test.opts.RemoteCertificate.ClientCertPath = clientCertPath
				}
				if test.clientKey != "" {
					clientKeyPath := fmt.Sprintf("%s/client.key", dir)
					err := os.WriteFile(clientKeyPath, []byte(test.clientKey), params.BeaconIoConfig().ReadWritePermissions)
					require.NoError(t, err, "Failed to write client key")
					test.opts.RemoteCertificate.ClientKeyPath = clientKeyPath
				}
			}
			_, err := NewKeymanager(context.Background(), &SetupConfig{Opts: test.opts, MaxMessageSize: 1})
			if test.err == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, test.err, err)
			}
		})
	}
}

func TestNewRemoteKeymanager_TlsDisabled(t *testing.T) {
	opts := &KeymanagerOpts{
		RemoteCertificate: &CertificateConfig{
			RequireTls: false,
		},
	}
	_, err := NewKeymanager(context.Background(), &SetupConfig{Opts: opts, MaxMessageSize: 1})
	assert.NoError(t, err)
}

func TestRemoteKeymanager_Sign(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := mock.NewMockRemoteSignerClient(ctrl)
	k := &Keymanager{
		client: m,
	}

	// Expect error handling to work.
	m.EXPECT().Sign(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(nil, errors.New("could not sign"))
	_, err := k.Sign(context.Background(), nil)
	require.ErrorContains(t, "could not sign", err)

	// Expected proper error handling for signing response statuses.
	m.EXPECT().Sign(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&validatorpb.SignResponse{
		Status: validatorpb.SignResponse_FAILED,
	}, nil /*err*/)
	_, err = k.Sign(context.Background(), nil)
	if err == nil {
		t.Fatal(err)
	}
	if err != ErrSigningFailed {
		t.Errorf("Expected %v, received %v", ErrSigningFailed, err)
	}
	m.EXPECT().Sign(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&validatorpb.SignResponse{
		Status: validatorpb.SignResponse_DENIED,
	}, nil /*err*/)
	_, err = k.Sign(context.Background(), nil)
	if err == nil {
		t.Fatal(err)
	}
	if err != ErrSigningDenied {
		t.Errorf("Expected %v, received %v", ErrSigningDenied, err)
	}

	// Expected signing success.
	randKey, err := bls.RandKey()
	require.NoError(t, err)
	data := []byte("hello-world")
	sig := randKey.Sign(data)
	m.EXPECT().Sign(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&validatorpb.SignResponse{
		Status:    validatorpb.SignResponse_SUCCEEDED,
		Signature: sig.Marshal(),
	}, nil /*err*/)
	resp, err := k.Sign(context.Background(), nil)
	require.NoError(t, err)
	assert.DeepEqual(t, sig.Marshal(), resp.Marshal())
}

func TestRemoteKeymanager_FetchValidatingPublicKeys(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := mock.NewMockRemoteSignerClient(ctrl)
	k := &Keymanager{
		client:              m,
		accountsChangedFeed: new(event.Feed),
	}

	// Expect error handling to work.
	m.EXPECT().ListValidatingPublicKeys(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(nil, errors.New("could not fetch keys"))
	_, err := k.FetchValidatingPublicKeys(context.Background())
	require.ErrorContains(t, "could not fetch keys", err)

	// Expect an empty response to return empty keys.
	m.EXPECT().ListValidatingPublicKeys(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&validatorpb.ListPublicKeysResponse{
		ValidatingPublicKeys: make([][]byte, 0),
	}, nil /*err*/)
	keys, err := k.FetchValidatingPublicKeys(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, len(keys), "Expected empty response")

	numKeys := 10
	pubKeys := make([][]byte, numKeys)
	for i := 0; i < numKeys; i++ {
		key := make([]byte, 48)
		copy(key, strconv.Itoa(i))
		pubKeys[i] = key
	}
	m.EXPECT().ListValidatingPublicKeys(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&validatorpb.ListPublicKeysResponse{
		ValidatingPublicKeys: pubKeys,
	}, nil /*err*/)
	keys, err = k.FetchValidatingPublicKeys(context.Background())
	require.NoError(t, err)
	rawKeys := make([][]byte, len(keys))
	for i := 0; i < len(rawKeys); i++ {
		rawKeys[i] = keys[i][:]
	}
	assert.DeepEqual(t, pubKeys, rawKeys)
}

func TestUnmarshalOptionsFile_DefaultRequireTls(t *testing.T) {
	optsWithoutTls := struct {
		RemoteCertificate struct {
			ClientCertPath string
			ClientKeyPath  string
			CACertPath     string
		}
	}{}
	var buffer bytes.Buffer
	b, err := json.Marshal(optsWithoutTls)
	require.NoError(t, err)
	_, err = buffer.Write(b)
	require.NoError(t, err)
	r := io.NopCloser(&buffer)

	opts, err := UnmarshalOptionsFile(r)
	assert.NoError(t, err)
	assert.Equal(t, true, opts.RemoteCertificate.RequireTls)
}

func TestReloadPublicKeys(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	m := mock.NewMockRemoteSignerClient(ctrl)

	k := &Keymanager{
		client:              m,
		accountsChangedFeed: new(event.Feed),
		orderedPubKeys:      [][fieldparams.BLSPubkeyLength]byte{bytesutil.ToBytes48([]byte("100"))},
	}

	// Add key
	m.EXPECT().ListValidatingPublicKeys(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&validatorpb.ListPublicKeysResponse{
		// Return keys in reverse order to verify ordering
		ValidatingPublicKeys: [][]byte{[]byte("200"), []byte("100")},
	}, nil /* err */)

	keys, err := k.ReloadPublicKeys(ctx)
	require.NoError(t, err)
	assert.DeepEqual(t, [][fieldparams.BLSPubkeyLength]byte{bytesutil.ToBytes48([]byte("100")), bytesutil.ToBytes48([]byte("200"))}, k.orderedPubKeys)
	assert.DeepEqual(t, keys, k.orderedPubKeys)
	assert.LogsContain(t, hook, keymanager.KeysReloaded)

	hook.Reset()

	// Remove key
	m.EXPECT().ListValidatingPublicKeys(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&validatorpb.ListPublicKeysResponse{
		ValidatingPublicKeys: [][]byte{[]byte("200")},
	}, nil /* err */)

	keys, err = k.ReloadPublicKeys(ctx)
	require.NoError(t, err)
	assert.DeepEqual(t, [][fieldparams.BLSPubkeyLength]byte{bytesutil.ToBytes48([]byte("200"))}, k.orderedPubKeys)
	assert.DeepEqual(t, keys, k.orderedPubKeys)
	assert.LogsContain(t, hook, keymanager.KeysReloaded)

	hook.Reset()

	// Change key
	m.EXPECT().ListValidatingPublicKeys(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&validatorpb.ListPublicKeysResponse{
		ValidatingPublicKeys: [][]byte{[]byte("300")},
	}, nil /* err */)

	keys, err = k.ReloadPublicKeys(ctx)
	require.NoError(t, err)
	assert.DeepEqual(t, [][fieldparams.BLSPubkeyLength]byte{bytesutil.ToBytes48([]byte("300"))}, k.orderedPubKeys)
	assert.DeepEqual(t, keys, k.orderedPubKeys)
	assert.LogsContain(t, hook, keymanager.KeysReloaded)

	hook.Reset()

	// No change
	m.EXPECT().ListValidatingPublicKeys(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&validatorpb.ListPublicKeysResponse{
		ValidatingPublicKeys: [][]byte{[]byte("300")},
	}, nil /* err */)

	keys, err = k.ReloadPublicKeys(ctx)
	require.NoError(t, err)
	assert.DeepEqual(t, [][fieldparams.BLSPubkeyLength]byte{bytesutil.ToBytes48([]byte("300"))}, k.orderedPubKeys)
	assert.DeepEqual(t, keys, k.orderedPubKeys)
	assert.LogsDoNotContain(t, hook, keymanager.KeysReloaded)
}
