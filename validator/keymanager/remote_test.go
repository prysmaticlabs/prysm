package keymanager_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
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

func TestNewRemoteWallet(t *testing.T) {
	tests := []struct {
		name       string
		opts       string
		clientCert string
		clientKey  string
		caCert     string
		err        string
	}{
		{
			name: "Empty",
			opts: ``,
			err:  "unexpected end of JSON input",
		},
		{
			name: "NoAccounts",
			opts: `{}`,
			err:  "at least one account specifier is required",
		},
		{
			name: "NoCertificates",
			opts: `{"accounts":["foo"]}`,
			err:  "certificates are required",
		},
		{
			name: "NoClientCertificate",
			opts: `{"accounts":["foo"],"certificates":{}}`,
			err:  "client certificate is required",
		},
		{
			name: "NoClientKey",
			opts: `{"accounts":["foo"],"certificates":{"client_cert":"foo"}}`,
			err:  "client key is required",
		},
		{
			name: "MissingClientKey",
			opts: `{"accounts":["foo"],"certificates":{"client_cert":"foo","client_key":"bar"}}`,
			err:  "failed to obtain client's certificate and/or key: open foo: no such file or directory",
		},
		{
			name:       "BadClientCert",
			clientCert: `bad`,
			clientKey:  validClientKey,
			opts:       `{"accounts":["foo"],"certificates":{"client_cert":"<<clientcert>>","client_key":"<<clientkey>>"}}`,
			err:        "failed to obtain client's certificate and/or key: tls: failed to find any PEM data in certificate input",
		},
		{
			name:       "BadClientKey",
			clientCert: validClientCert,
			clientKey:  `bad`,
			opts:       `{"accounts":["foo"],"certificates":{"client_cert":"<<clientcert>>","client_key":"<<clientkey>>"}}`,
			err:        "failed to obtain client's certificate and/or key: tls: failed to find any PEM data in key input",
		},
		{
			name:       "MissingCACert",
			clientCert: validClientCert,
			clientKey:  validClientKey,
			opts:       `{"accounts":["foo"],"certificates":{"client_cert":"<<clientcert>>","client_key":"<<clientkey>>","ca_cert":"bad"}}`,
			err:        "failed to obtain server's CA certificate: open bad: no such file or directory",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			if test.caCert != "" || test.clientCert != "" || test.clientKey != "" {
				dir := fmt.Sprintf("%s/%s", testutil.TempDir(), test.name)
				if err := os.MkdirAll(dir, 0777); err != nil {
					t.Fatalf(err.Error())
				}
				if test.caCert != "" {
					caCertPath := fmt.Sprintf("%s/ca.crt", dir)
					if err := ioutil.WriteFile(caCertPath, []byte(test.caCert), params.BeaconIoConfig().FilePermission); err != nil {
						t.Fatalf("Failed to write CA certificate: %v", err)
					}
					test.opts = strings.ReplaceAll(test.opts, "<<cacert>>", caCertPath)
				}
				if test.clientCert != "" {
					clientCertPath := fmt.Sprintf("%s/client.crt", dir)
					if err := ioutil.WriteFile(clientCertPath, []byte(test.clientCert), params.BeaconIoConfig().FilePermission); err != nil {
						t.Fatalf("Failed to write client certificate: %v", err)
					}
					test.opts = strings.ReplaceAll(test.opts, "<<clientcert>>", clientCertPath)
				}
				if test.clientKey != "" {
					clientKeyPath := fmt.Sprintf("%s/client.key", dir)
					if err := ioutil.WriteFile(clientKeyPath, []byte(test.clientKey), params.BeaconIoConfig().FilePermission); err != nil {
						t.Fatalf("Failed to write client key: %v", err)
					}
					test.opts = strings.ReplaceAll(test.opts, "<<clientkey>>", clientKeyPath)
				}
			}

			_, _, err := keymanager.NewRemoteWallet(test.opts)
			if test.err == "" {
				if err != nil {
					t.Fatalf("Received unexpected error: %v", err.Error())
				}
			} else {
				if err == nil {
					t.Fatal("Did not received an error")
				}
				if err.Error() != test.err {
					t.Fatalf("Did not received expected error: expected %v, received %v", test.err, err.Error())
				}
			}
		})
	}
}
