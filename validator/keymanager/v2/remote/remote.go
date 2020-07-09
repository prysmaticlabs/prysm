package remote

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"io/ioutil"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	validatorpb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var log = logrus.WithField("prefix", "remote-keymanager-v2")

// Config for a remote keymanager.
type Config struct {
	RemoteCertificate *Certificate `json:"remote_cert"`
	RemoteAddr        string       `json:"remote_address"`
}

type Certificate struct {
	ClientCertPath string `json:"crt_path"`
	ClientKeyPath  string `json:"key_path"`
	CACertPath     string `json:"ca_crt_path"`
}

// Keymanager implementation using remote signing keys via gRPC.
type Keymanager struct {
	cfg              *Config
	client           validatorpb.RemoteSignerClient
	accountsByPubkey map[[48]byte]string
}

// NewKeymanager instantiates a new direct keymanager from configuration options.
func NewKeymanager(ctx context.Context, maxMessageSize int, cfg *Config) (*Keymanager, error) {
	// Load the client certificates.
	if cfg.RemoteCertificate == nil {
		return nil, errors.New("certificates are required")
	}
	if cfg.RemoteCertificate.ClientCertPath == "" {
		return nil, errors.New("client certificate is required")
	}
	if cfg.RemoteCertificate.ClientKeyPath == "" {
		return nil, errors.New("client key is required")
	}
	clientPair, err := tls.LoadX509KeyPair(cfg.RemoteCertificate.ClientCertPath, cfg.RemoteCertificate.ClientKeyPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain client's certificate and/or key")
	}

	// Load the CA for the server certificate if present.
	cp := x509.NewCertPool()
	if cfg.RemoteCertificate.CACertPath != "" {
		serverCA, err := ioutil.ReadFile(cfg.RemoteCertificate.CACertPath)
		if err != nil {
			return nil, errors.Wrap(err, "failed to obtain server's CA certificate")
		}
		if !cp.AppendCertsFromPEM(serverCA) {
			return nil, errors.Wrap(err, "failed to add server's CA certificate to pool")
		}
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{clientPair},
		RootCAs:      cp,
	}
	clientCreds := credentials.NewTLS(tlsCfg)

	grpcOpts := []grpc.DialOption{
		// Require TLS with client certificate.
		grpc.WithTransportCredentials(clientCreds),
		// Receive large messages without erroring.
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxMessageSize)),
	}
	conn, err := grpc.Dial(cfg.RemoteAddr, grpcOpts...)
	if err != nil {
		return nil, errors.New("failed to connect to remote wallet")
	}
	client := validatorpb.NewRemoteSignerClient(conn)
	k := &Keymanager{
		cfg:              cfg,
		client:           client,
		accountsByPubkey: make(map[[48]byte]string),
	}
	return k, nil
}

// UnmarshalConfigFile attempts to JSON unmarshal a direct keymanager
// configuration file into the *Config{} struct.
func UnmarshalConfigFile(r io.ReadCloser) (*Config, error) {
	enc, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := r.Close(); err != nil {
			log.Errorf("Could not close keymanager config file: %v", err)
		}
	}()
	cfg := &Config{}
	if err := json.Unmarshal(enc, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// CreateAccount based on the keymanager's logic. Returns the account name.
func (k *Keymanager) CreateAccount(ctx context.Context, password string) (string, error) {
	return "", errors.New("a remote validator account cannot be created from the client")
}

// MarshalConfigFile for the keymanager's options.
func (k *Keymanager) MarshalConfigFile(ctx context.Context) ([]byte, error) {
	return json.MarshalIndent(k.cfg, "", "\t")
}

// FetchValidatingPublicKeys fetches the list of public keys that should be used to validate with.
func (k *Keymanager) FetchValidatingPublicKeys(ctx context.Context) ([][48]byte, error) {
	resp, err := k.client.ListValidatingPublicKeys(ctx, &ptypes.Empty{})
	if err != nil {
		return nil, errors.Wrap(err, "could not list accounts from remote server")
	}
	pubKeys := make([][48]byte, len(resp.ValidatingPublicKeys))
	for i := range resp.ValidatingPublicKeys {
		pubKeys[i] = bytesutil.ToBytes48(resp.ValidatingPublicKeys[i])
	}
	return pubKeys, nil
}

// Sign signs a message for a validator key via a gRPC request.
func (k *Keymanager) Sign(ctx context.Context, req *validatorpb.SignRequest) (bls.Signature, error) {
	resp, err := k.client.Sign(ctx, req)
	if err != nil {
		return nil, err
	}
	switch resp.Status {
	case validatorpb.SignResponse_DENIED:
		return nil, errors.Wrap(err, "signing request denied by remote server")
	case validatorpb.SignResponse_FAILED:
		return nil, errors.Wrap(err, "signing request failed")
	}
	return bls.SignatureFromBytes(resp.Signature)
}
