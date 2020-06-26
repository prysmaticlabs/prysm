package rpc

import (
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/rand"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
)

const (
	rsaBits  = 2048
	validFor = 365 * 24 * time.Hour
)

var (
	selfSignedCertName    = "beacon.pem"
	selfSignedCertKeyName = "key.pem"
)

// Generates self-signed certificates at a datadir path. This function
// returns the paths of the cert.pem and key.pem files that
// were generated as a result.
func generateSelfSignedCerts(datadir string) (string, string, error) {
	priv, err := rsa.GenerateKey(rand.NewGenerator(), rsaBits)
	if err != nil {
		return "", "", errors.Wrap(err, "nailed to generate private key")
	}

	notBefore := roughtime.Now()
	notAfter := notBefore.Add(validFor)

	serialNumber := big.NewInt(int64(rand.NewGenerator().Int() % 128))
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Prysm"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	template.IPAddresses = append(template.IPAddresses, net.ParseIP("127.0.0.1"))

	template.IsCA = true
	template.KeyUsage |= x509.KeyUsageCertSign

	derBytes, err := x509.CreateCertificate(rand.NewGenerator(), &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to create x509 certificate")
	}

	certPath := path.Join(datadir, selfSignedCertName)
	certKeyPath := path.Join(datadir, selfSignedCertKeyName)
	certOut, err := os.Create(certPath)
	if err != nil {
		return "", "", errors.Wrapf(err, "failed to open %s for writing", certPath)
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return "", "", errors.Wrapf(err, "failed to write data to %s", certPath)
	}
	if err := certOut.Close(); err != nil {
		return "", "", errors.Wrapf(err, "error closing write buffer: %s", certPath)
	}
	log.WithField("certPath", certPath).Info("Wrote self-signed certificate file")

	keyOut, err := os.OpenFile(certKeyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return "", "", errors.Wrapf(err, "failed to open %s for writing", certKeyPath)
	}
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return "", "", errors.Wrap(err, "unable to marshal private key")
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		return "", "", errors.Wrapf(err, "failed to write data to %s", certKeyPath)
	}
	if err := keyOut.Close(); err != nil {
		return "", "", errors.Wrapf(err, "error closing write buffer: %s", certKeyPath)
	}
	log.WithField("certKeyPath", certKeyPath).Info("Wrote self-signed certificate key file")
	return certPath, certKeyPath, nil
}
